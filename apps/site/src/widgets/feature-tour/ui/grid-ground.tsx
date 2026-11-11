/**
 * The one photograph behind every highlight card (public/backdrops, credited
 * on /credits). Each card renders it and clips it to its own shape; from
 * `lg` up the image is sized to the whole grid, so the cards read as windows
 * onto a single frame. Decorative: it carries no alt.
 */
export function GridGround() {
  return (
    // eslint-disable-next-line @next/next/no-img-element -- static export, pre-sized webp
    <img
      src="/backdrops/coastal-1600.webp"
      srcSet="/backdrops/coastal-960.webp 960w, /backdrops/coastal-1600.webp 1600w"
      sizes="(min-width: 1024px) 80rem, 100vw"
      alt=""
      aria-hidden="true"
      loading="lazy"
      className="absolute inset-0 -z-10 size-full object-cover"
    />
  );
}
