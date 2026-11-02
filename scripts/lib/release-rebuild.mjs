// Same-version rebuilding is an explicit operator action, never a default retry.
export function releaseRebuildPlan(config, previous, expectedCommit, channels) {
  if (!/^[a-f0-9]{40}$/.test(expectedCommit ?? '') || previous?.source?.commit !== expectedCommit) {
    throw new Error('Staged release does not match the expected replacement commit');
  }
  if (!config.version.includes('-') || previous.schema !== 'astrafoundry.release.v1' ||
      previous.version !== config.version || previous.product !== config.release.product ||
      previous.source.repository !== config.release.repository || previous.source.tag !== `v${config.version}`) {
    throw new Error('Only the matching prerelease can be rebuilt');
  }
  for (const channel of ['stable', 'beta']) {
    const index = channels[channel];
    if (index?.channel !== channel || !index.targets || typeof index.targets !== 'object' || Array.isArray(index.targets)) {
      throw new Error(`Invalid ${channel} channel response`);
    }
    for (const target of Object.values(index.targets)) {
      if (typeof target?.version !== 'string') throw new Error(`Invalid ${channel} target`);
      if (target.version === config.version) throw new Error('Refusing to rebuild a release offered by an update channel');
    }
  }
  const files = Object.values(previous.targets).flatMap(target => [target.installer, target.update, ...target.deltas]);
  const names = files.map(file => file.name);
  if (names.some(name => typeof name !== 'string' || !/^[A-Za-z0-9._-]+$/.test(name) || !name.includes(config.version)) ||
      JSON.stringify(previous.documents) !== JSON.stringify(['SHA256SUMS', 'RELEASE-STATUS.md'])) {
    throw new Error('Unexpected release backup paths');
  }
  const prefix = `releases/${config.version}/replaced/${expectedCommit}`;
  return {
    recordKey: `${prefix}/release.json`,
    copies: [
      ...names.map(name => ({ source: `files/${name}`, destination: `${prefix}/files/${name}` })),
      ...previous.documents.map(name => ({ source: `releases/${config.version}/${name}`, destination: `${prefix}/${name}` })),
    ],
  };
}
