import * as React from "react"
import type * as THREE from "three"

import { cn } from "@/lib/utils"

// The Go gopher, after Renée French's design (CC BY 4.0), riding a rocket,
// drawn as anime: cel shading in three tones, ink outlines, a hard rim
// light, speed lines and sparkles. Three.js loads on its own, after the
// page, and the scene draws only while it is on screen; a reader who asked
// for less motion gets one still frame.
export function GopherRocket({ className }: { className?: string }) {
  const ref = React.useRef<HTMLDivElement>(null)

  React.useEffect(() => {
    const el = ref.current
    if (!el) return
    let cleanup = () => {}
    let cancelled = false
    import("three").then((three) => {
      if (!cancelled) cleanup = mount(three, el)
    })
    return () => {
      cancelled = true
      cleanup()
    }
  }, [])

  return <div ref={ref} aria-hidden className={cn("gopher-rocket", className)} />
}

const ink = 0x1a2233
const fur = 0x5ccbea
const tan = 0xf8d8a8
const rocketRed = 0xff4d68
// The wordmark's "!" colors, and gold, for the sparkles.
const sparkleColors = [0x5aa9c4, 0xe9708f, 0x4fae5d, 0xe0b23a, 0x8a83bd, 0xffc93c]

type Line = "thick" | "thin" | false

function mount(T: typeof THREE, el: HTMLElement) {
  const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches
  const renderer = new T.WebGLRenderer({ antialias: true, alpha: true })
  renderer.setPixelRatio(Math.min(devicePixelRatio, 2))
  renderer.domElement.style.display = "block"
  el.append(renderer.domElement)

  const scene = new T.Scene()
  const camera = new T.PerspectiveCamera(32, 1, 0.1, 100)
  camera.position.set(0.25, 0.1, 12.5)

  scene.add(new T.AmbientLight(0xffffff, 1.0))
  const key = new T.DirectionalLight(0xffffff, 2.7)
  key.position.set(4, 6, 5)
  scene.add(key)

  const materials: THREE.Material[] = []
  const geometries: THREE.BufferGeometry[] = []
  const textures: THREE.Texture[] = []
  const geo = <G extends THREE.BufferGeometry>(g: G) => (geometries.push(g), g)

  // Cel shading: the light falls into three flat tones, and a hard rim of
  // warm light catches the edges that face up and to the left.
  const tones = new T.DataTexture(new Uint8Array([78, 165, 255]), 3, 1, T.RedFormat)
  tones.minFilter = T.NearestFilter
  tones.magFilter = T.NearestFilter
  tones.needsUpdate = true
  textures.push(tones)
  const cel = (color: number, rim = 0.35) => {
    const m = new T.MeshToonMaterial({ color, gradientMap: tones })
    m.onBeforeCompile = (s) => {
      s.fragmentShader = s.fragmentShader.replace(
        "#include <opaque_fragment>",
        `float facing = dot(normal, normalize(vViewPosition));
        float rim = step(facing, 0.36) * step(0.1, dot(normal, normalize(vec3(-0.55, 0.65, 0.3))));
        outgoingLight = mix(outgoingLight, vec3(1.0, 0.97, 0.88), rim * ${rim.toFixed(2)});
        #include <opaque_fragment>`,
      )
    }
    m.customProgramCacheKey = () => `cel-${rim}`
    materials.push(m)
    return m
  }
  const flat = (color: number, extra: THREE.MeshBasicMaterialParameters = {}) => {
    const m = new T.MeshBasicMaterial({ color, ...extra })
    materials.push(m)
    return m
  }
  // Ink outlines: the back faces, pushed out along their normals in view
  // space, so a line is as thick on a pupil as on the hull.
  const outline = (width: number) => {
    const m = new T.MeshBasicMaterial({ color: ink, side: T.BackSide })
    m.onBeforeCompile = (s) => {
      s.vertexShader = s.vertexShader.replace(
        "#include <project_vertex>",
        `#include <project_vertex>
        mvPosition.xyz += normalize(normalMatrix * normal) * ${width.toFixed(3)};
        gl_Position = projectionMatrix * mvPosition;`,
      )
    }
    m.customProgramCacheKey = () => `outline-${width}`
    materials.push(m)
    return m
  }
  const lines = { thick: outline(0.03), thin: outline(0.016) }

  const mesh = (
    g: THREE.BufferGeometry,
    m: THREE.Material,
    parent: THREE.Object3D,
    pos: [number, number, number],
    scale?: [number, number, number],
    line: Line = "thick",
  ) => {
    const o = new T.Mesh(g, m)
    o.position.set(...pos)
    if (scale) o.scale.set(...scale)
    if (line) o.add(new T.Mesh(g, lines[line]))
    parent.add(o)
    return o
  }

  // A pale disc behind the rider, as the sun in a key frame.
  const discMat = flat(0xdff3fa, { transparent: true, depthWrite: false })
  const disc = new T.Mesh(geo(new T.CircleGeometry(3.1, 96)), discMat)
  disc.position.set(0.6, 0.5, -6)
  scene.add(disc)

  // The rider: the rocket lies along +x, the gopher sits on top of it.
  const rider = new T.Group()
  scene.add(rider)

  // Rocket, built along +y, then laid down.
  const rocket = new T.Group()
  rocket.rotation.z = -Math.PI / 2
  rider.add(rocket)
  const hull = cel(0xf6f7fb)
  const red = cel(rocketRed)
  const steel = cel(0x9aa6b4)
  const dark = cel(0x39404c, 0.2)
  const v = (x: number, y: number) => new T.Vector2(x, y)
  mesh(geo(new T.LatheGeometry([v(0.001, -1), v(0.44, -1), v(0.52, -0.65), v(0.56, -0.1), v(0.56, 0.5), v(0.51, 1)], 48)), hull, rocket, [0, 0, 0])
  mesh(geo(new T.LatheGeometry([v(0.51, 1), v(0.47, 1.25), v(0.38, 1.55), v(0.24, 1.82), v(0.08, 1.98), v(0.001, 2)], 48)), red, rocket, [0, 0, 0])
  const band = mesh(geo(new T.TorusGeometry(0.53, 0.04, 12, 48)), steel, rocket, [0, 1, 0], undefined, "thin")
  band.rotation.x = Math.PI / 2
  const stripe = mesh(geo(new T.CylinderGeometry(0.565, 0.565, 0.16, 48, 1, true)), red, rocket, [0, -0.45, 0], undefined, false)
  stripe.material = red
  mesh(geo(new T.CylinderGeometry(0.3, 0.4, 0.32, 32)), dark, rocket, [0, -1.14, 0])
  // Porthole, on the side that faces the camera, with a glint.
  mesh(geo(new T.TorusGeometry(0.17, 0.045, 12, 32)), steel, rocket, [0, 0.55, 0.53], undefined, "thin")
  mesh(geo(new T.CircleGeometry(0.17, 32)), cel(0x2fb6e3, 0), rocket, [0, 0.55, 0.52], undefined, false)
  const shine = flat(0xffffff)
  mesh(geo(new T.CircleGeometry(0.05, 16)), shine, rocket, [-0.06, 0.62, 0.535], [1, 1.3, 1], false)
  // Fins.
  const finShape = new T.Shape()
  finShape.moveTo(0, -0.05)
  finShape.quadraticCurveTo(0.4, -0.35, 0.48, -1.15)
  finShape.lineTo(0.3, -1.05)
  finShape.lineTo(0, -0.95)
  const finGeo = geo(new T.ExtrudeGeometry(finShape, { depth: 0.06, bevelEnabled: true, bevelSize: 0.02, bevelThickness: 0.02, bevelSegments: 2 }))
  finGeo.translate(0, 0, -0.03)
  for (let i = 0; i < 3; i++) {
    const pivot = new T.Group()
    pivot.rotation.y = (i * Math.PI * 2) / 3 + Math.PI / 2
    mesh(finGeo, red, pivot, [0.46, 0, 0])
    rocket.add(pivot)
  }
  // Flame: three flat layers, drawn over one another so the core shows
  // through without blending. They draw last, after the disc, the speed
  // lines and the sparkles, which write no depth and would cover them.
  const flameGeo = geo(new T.ConeGeometry(1, 1, 24, 1, true))
  flameGeo.rotateX(Math.PI)
  flameGeo.translate(0, -0.5, 0)
  const flames = [0xff5a36, 0xffa531, 0xfff1a8].map((color, i) => {
    const o = new T.Mesh(flameGeo, flat(color, { side: T.DoubleSide, transparent: true, depthWrite: false }))
    o.position.set(0, -1.28, 0)
    o.renderOrder = 10 + i
    rocket.add(o)
    return o
  })

  // Gopher, facing the camera, a little toward where it is going.
  const gopher = new T.Group()
  gopher.position.set(-0.05, 0.42, 0)
  gopher.rotation.y = 0.3
  gopher.scale.setScalar(1.2)
  rider.add(gopher)
  const furMat = cel(fur)
  const tanMat = cel(tan)
  const eyeWhite = cel(0xffffff, 0)
  const iris = cel(0x2a5fd0, 0)
  const pupil = flat(0x0c1222)
  const sphere = geo(new T.SphereGeometry(1, 48, 32))
  mesh(sphere, furMat, gopher, [0, 0.62, 0], [0.62, 0.7, 0.56])
  // Ears.
  mesh(sphere, furMat, gopher, [-0.44, 1.14, 0], [0.11, 0.11, 0.07])
  mesh(sphere, furMat, gopher, [0.44, 1.14, 0], [0.11, 0.11, 0.07])
  // Eyes, as anime draws them: a deep blue iris, a pupil, and two glints.
  // They blink, so each one is a group scaled on y.
  const eyes: THREE.Group[] = []
  for (const side of [-1, 1]) {
    const eye = new T.Group()
    eye.position.set(side * 0.23, 0.96, 0.36)
    eye.rotation.y = side * 0.25
    mesh(sphere, eyeWhite, eye, [0, 0, 0], [0.22, 0.24, 0.16])
    const look = side * -0.03
    mesh(sphere, iris, eye, [look, -0.025, 0.112], [0.115, 0.135, 0.06], "thin")
    mesh(sphere, pupil, eye, [look, -0.035, 0.152], [0.058, 0.072, 0.034], false)
    mesh(sphere, shine, eye, [look - 0.04, 0.035, 0.172], [0.04, 0.042, 0.014], false)
    mesh(sphere, shine, eye, [look + 0.035, -0.075, 0.164], [0.018, 0.018, 0.01], false)
    gopher.add(eye)
    eyes.push(eye)
    // Blush.
    const blush = mesh(sphere, flat(0xff8fa8, { transparent: true, opacity: 0.6, depthWrite: false }), gopher, [side * 0.37, 0.71, 0.44], [0.09, 0.045, 0.02], false)
    blush.rotation.y = side * 0.68
  }
  // Muzzle, nose and teeth.
  mesh(sphere, tanMat, gopher, [0, 0.7, 0.5], [0.2, 0.13, 0.12])
  mesh(sphere, pupil, gopher, [0, 0.79, 0.6], [0.075, 0.05, 0.045], "thin")
  mesh(sphere, shine, gopher, [-0.025, 0.805, 0.64], [0.02, 0.012, 0.008], false)
  const tooth = geo(new T.BoxGeometry(0.075, 0.1, 0.03))
  mesh(tooth, eyeWhite, gopher, [-0.042, 0.57, 0.585], undefined, "thin")
  mesh(tooth, eyeWhite, gopher, [0.042, 0.57, 0.585], undefined, "thin")
  // Arms hang from the shoulders onto the hull and end in round paws; the
  // near one waves now and then.
  const armRest = { x: -0.7, z: 0.25 }
  const limb = geo(new T.CapsuleGeometry(0.08, 0.22, 6, 12))
  const arm = (side: number) => {
    const a = new T.Group()
    a.position.set(side * 0.5, 0.44, 0.2)
    a.rotation.set(armRest.x, 0, -side * armRest.z)
    mesh(limb, tanMat, a, [0, -0.15, 0])
    mesh(sphere, tanMat, a, [0, -0.3, 0.01], [0.1, 0.08, 0.1])
    gopher.add(a)
    return a
  }
  arm(-1)
  const armR = arm(1)
  // Legs dangle over the side of the rocket.
  for (const side of [-1, 1]) {
    const leg = new T.Group()
    leg.position.set(side * 0.24, 0.12, 0.38)
    leg.rotation.set(-0.5, 0, side * 0.15)
    mesh(limb, tanMat, leg, [0, -0.14, 0])
    mesh(sphere, tanMat, leg, [0, -0.3, 0.06], [0.1, 0.07, 0.14])
    gopher.add(leg)
  }

  // The rider points up and to the right.
  const tilt = 0.5
  const axis = new T.Vector3(Math.cos(tilt), Math.sin(tilt), 0)
  const across = new T.Vector3(-axis.y, axis.x, 0)
  const flight = new T.Quaternion().setFromEuler(new T.Euler(0, 0, tilt))

  // Smoke: cel-shaded puffs with ink lines, which swell and shrink away
  // rather than fade, as anime draws them. They stay behind in the scene
  // as the rocket moves on.
  const puffGeo = geo(new T.IcosahedronGeometry(1, 3))
  const puffMat = cel(0xffffff, 0.25)
  const puffs = Array.from({ length: 40 }, () => {
    const o = mesh(puffGeo, puffMat, scene, [0, 0, 0])
    o.visible = false
    return { o, vel: new T.Vector3(), age: 0, life: 1, size: 0.2 }
  })
  let nextPuff = 0
  const nozzle = new T.Vector3()
  const back = new T.Vector3()

  // Speed lines stream past along the line of flight, tapered at both ends.
  const taper = document.createElement("canvas")
  taper.width = 128
  taper.height = 1
  const tctx = taper.getContext("2d")!
  const grad = tctx.createLinearGradient(0, 0, 128, 0)
  grad.addColorStop(0, "rgba(255,255,255,0)")
  grad.addColorStop(0.6, "rgba(255,255,255,1)")
  grad.addColorStop(1, "rgba(255,255,255,0)")
  tctx.fillStyle = grad
  tctx.fillRect(0, 0, 128, 1)
  const taperTex = new T.CanvasTexture(taper)
  textures.push(taperTex)
  const streakMat = flat(0xffffff, { map: taperTex, transparent: true, depthWrite: false })
  const streakCount = 44
  const streaks = new T.InstancedMesh(geo(new T.PlaneGeometry(1, 1)), streakMat, streakCount)
  streaks.frustumCulled = false
  const streak = Array.from({ length: streakCount }, (_, i) => ({
    u: (Math.random() - 0.5) * 20,
    v: (Math.random() - 0.5) * 11,
    // A few pass in front of the rider, the rest behind.
    z: i < 5 ? 1.5 + Math.random() : -1.5 - Math.random() * 4,
    len: 1.2 + Math.random() * 3.4,
    width: 0.018 + Math.random() * 0.035,
    speed: 7 + Math.random() * 9,
  }))
  scene.add(streaks)
  const m4 = new T.Matrix4()
  const p3 = new T.Vector3()
  const s3 = new T.Vector3()

  // Sparkles: four-pointed stars that twinkle as they drift past.
  const star = new T.Shape()
  star.moveTo(0, 1)
  star.quadraticCurveTo(0.08, 0.08, 1, 0)
  star.quadraticCurveTo(0.08, -0.08, 0, -1)
  star.quadraticCurveTo(-0.08, -0.08, -1, 0)
  star.quadraticCurveTo(-0.08, 0.08, 0, 1)
  const starGeo = geo(new T.ShapeGeometry(star, 10))
  const sparkleMats = sparkleColors.map((c) => flat(c, { transparent: true, depthWrite: false }))
  const sparkles = Array.from({ length: 22 }, (_, i) => {
    const o = new T.Mesh(starGeo, sparkleMats[i % sparkleMats.length])
    scene.add(o)
    return {
      o,
      u: (Math.random() - 0.5) * 14,
      v: (Math.random() - 0.5) * 9,
      z: -2.5 + Math.random() * 3,
      size: 0.14 + Math.random() * 0.2,
      phase: Math.random() * Math.PI * 2,
      rate: 2 + Math.random() * 3,
    }
  })

  const paint = () => {
    const isDark = document.documentElement.classList.contains("dark")
    discMat.color.setHex(isDark ? 0x0f2c3a : 0xdff3fa)
    streakMat.color.setHex(isDark ? 0x9fdcf0 : 0x00add8)
    streakMat.opacity = isDark ? 0.32 : 0.28
    puffMat.color.setHex(isDark ? 0xdce6ee : 0xffffff)
  }
  paint()
  const themeObserver = new MutationObserver(() => {
    paint()
    if (reduced) render(still)
  })
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] })

  // The pointer leans the rider a little.
  const pointer = { x: 0, y: 0, tx: 0, ty: 0 }
  const onPointer = (e: PointerEvent) => {
    pointer.tx = (e.clientX / innerWidth) * 2 - 1
    pointer.ty = (e.clientY / innerHeight) * 2 - 1
  }
  if (!reduced) addEventListener("pointermove", onPointer, { passive: true })

  const resize = () => {
    const w = el.clientWidth
    const h = el.clientHeight
    if (!w || !h) return
    renderer.setSize(w, h, false)
    renderer.domElement.style.width = "100%"
    renderer.domElement.style.height = "100%"
    camera.aspect = w / h
    camera.updateProjectionMatrix()
    if (reduced) render(still)
  }
  const resizeObserver = new ResizeObserver(resize)
  resizeObserver.observe(el)

  const still = 2.2
  let last = 0
  let puffClock = 0
  const render = (t: number, dt = 0) => {
    // Bob and sway.
    pointer.x += (pointer.tx - pointer.x) * Math.min(1, dt * 3)
    pointer.y += (pointer.ty - pointer.y) * Math.min(1, dt * 3)
    rider.position.set(Math.sin(t * 0.7) * 0.08, Math.sin(t * 1.6) * 0.14, 0)
    rider.rotation.set(pointer.y * 0.18 + Math.sin(t * 0.9) * 0.05, pointer.x * 0.35 - 0.25, tilt + Math.sin(t * 1.1) * 0.05)
    disc.position.x = 0.6 + pointer.x * -0.15
    disc.position.y = 0.5 + pointer.y * 0.1

    // Blink every few seconds, wave now and then.
    const blink = t % 3.7 < 0.12 ? 0.1 : 1
    for (const eye of eyes) eye.scale.y = blink
    const wave = t % 7
    if (wave < 1.6) {
      const k = Math.sin((wave / 1.6) * Math.PI)
      armR.rotation.set(armRest.x * (1 - k), 0, -armRest.z + k * 2.85 + Math.sin(wave * 14) * 0.35 * k)
    } else armR.rotation.set(armRest.x, 0, -armRest.z)

    // Flicker, each layer on its own beat.
    const f = (a: number, b: number) => 1 + Math.sin(t * a) * 0.09 + Math.sin(t * b) * 0.06
    const [outer, mid, core] = flames
    outer.scale.set(0.36, 1.6 * f(37, 23), 0.36)
    mid.scale.set(0.26, 1.15 * f(41, 19), 0.26)
    core.scale.set(0.15, 0.72 * f(29, 47), 0.15)

    // Speed lines.
    for (let i = 0; i < streakCount; i++) {
      const s = streak[i]
      s.u -= s.speed * dt
      if (s.u < -11) {
        s.u += 22
        s.v = (Math.random() - 0.5) * 11
      }
      p3.copy(axis).multiplyScalar(s.u).addScaledVector(across, s.v)
      p3.z = s.z
      m4.compose(p3, flight, s3.set(s.len, s.width, 1))
      streaks.setMatrixAt(i, m4)
    }
    streaks.instanceMatrix.needsUpdate = true

    // Sparkles.
    for (const s of sparkles) {
      s.u -= dt * 1.4
      if (s.u < -7) {
        s.u += 14
        s.v = (Math.random() - 0.5) * 9
      }
      s.o.position.copy(axis).multiplyScalar(s.u).addScaledVector(across, s.v)
      s.o.position.z = s.z
      const tw = Math.max(0, Math.sin(t * s.rate + s.phase))
      s.o.scale.setScalar(s.size * (0.15 + 0.85 * tw ** 2))
      s.o.rotation.z = t * 0.4 + s.phase
    }

    // Smoke.
    rider.updateMatrixWorld()
    puffClock += dt
    while (puffClock > 0.05) {
      puffClock -= 0.05
      const p = puffs[nextPuff++ % puffs.length]
      nozzle.set((Math.random() - 0.5) * 0.18, -2.15, (Math.random() - 0.5) * 0.18)
      rocket.localToWorld(nozzle)
      p.o.position.copy(nozzle)
      back.set(0, -1, 0).transformDirection(rocket.matrixWorld)
      p.vel.copy(back).multiplyScalar(0.9 + Math.random() * 0.8).addScaledVector(axis, -0.9)
      p.vel.x += (Math.random() - 0.5) * 0.6
      p.vel.y += (Math.random() - 0.5) * 0.6
      p.age = 0
      p.life = 1.3 + Math.random() * 0.8
      p.size = 0.22 + Math.random() * 0.18
      p.o.visible = true
    }
    for (const p of puffs) {
      if (!p.o.visible) continue
      p.age += dt
      const k = p.age / p.life
      if (k >= 1) {
        p.o.visible = false
        continue
      }
      p.o.position.addScaledVector(p.vel, dt)
      p.vel.multiplyScalar(1 - dt * 1.2)
      // Swell quickly, then shrink to nothing.
      p.o.scale.setScalar(p.size * Math.min(1, k * 6) * (1 + k * 1.8) * (1 - k ** 3))
    }

    renderer.render(scene, camera)
  }

  let frame = 0
  const loop = (now: number) => {
    frame = requestAnimationFrame(loop)
    const t = now / 1000
    const dt = last ? Math.min(t - last, 0.05) : 0
    last = t
    render(t, dt)
  }
  const intersection = new IntersectionObserver(([entry]) => {
    if (reduced) return
    cancelAnimationFrame(frame)
    last = 0
    if (entry.isIntersecting) frame = requestAnimationFrame(loop)
  })
  intersection.observe(el)

  if (reduced) {
    // Let the smoke build up behind the rocket, then hold still.
    for (let i = 0; i < 60; i++) render(still - 1.8 + i * 0.03, 0.03)
  }
  resize()

  return () => {
    cancelAnimationFrame(frame)
    intersection.disconnect()
    resizeObserver.disconnect()
    themeObserver.disconnect()
    removeEventListener("pointermove", onPointer)
    for (const g of geometries) g.dispose()
    for (const m of materials) m.dispose()
    for (const x of textures) x.dispose()
    renderer.dispose()
    renderer.domElement.remove()
  }
}
