"use client";

import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import { Pause, Play } from "lucide-react";

const motionQuery = "(prefers-reduced-motion: reduce)";
function subscribeMotion(callback: () => void) {
  const query = window.matchMedia(motionQuery);
  query.addEventListener("change", callback);
  return () => query.removeEventListener("change", callback);
}

export function HeroLattice() {
  const host = useRef<HTMLDivElement>(null);
  const syncPlayback = useRef<(() => void) | null>(null);
  const [ready, setReady] = useState(false);
  const [playing, setPlaying] = useState(true);
  const reducedMotion = useSyncExternalStore(subscribeMotion, () => window.matchMedia(motionQuery).matches, () => true);
  const paused = useRef(true);

  useEffect(() => {
    paused.current = !playing || reducedMotion;
    syncPlayback.current?.();
  }, [playing, reducedMotion]);

  useEffect(() => {
    const element = host.current;
    if (!element) return;
    let cancelled = false;
    let initializing = false;
    let visible = false;
    let cleanupRenderer = () => {};

    async function initialize() {
      if (initializing) return;
      initializing = true;
      try {
        const THREE = await import("three");
        if (cancelled) return;
        const renderer = new THREE.WebGLRenderer({ alpha: true, antialias: true, powerPreference: "low-power" });
        renderer.setClearColor(0x0a0a0a, 0);
        renderer.setPixelRatio(Math.min(window.devicePixelRatio, 1.5));
        renderer.outputColorSpace = THREE.SRGBColorSpace;
        renderer.domElement.setAttribute("aria-hidden", "true");
        element!.appendChild(renderer.domElement);

        const scene = new THREE.Scene();
        const camera = new THREE.OrthographicCamera(-3.6, 3.6, 3.3, -3.3, 0.1, 30);
        camera.position.z = 10;
        const lattice = new THREE.Group();
        scene.add(lattice);
        const geometries: InstanceType<typeof THREE.BufferGeometry>[] = [];
        const materials: InstanceType<typeof THREE.Material>[] = [];
        const lines: InstanceType<typeof THREE.Group>[] = [];

        const shape = new THREE.Shape();
        const half = 1.8;
        const radius = 0.3;
        shape.moveTo(-half + radius, -half);
        shape.lineTo(half - radius, -half);
        shape.quadraticCurveTo(half, -half, half, -half + radius);
        shape.lineTo(half, half - radius);
        shape.quadraticCurveTo(half, half, half - radius, half);
        shape.lineTo(-half + radius, half);
        shape.quadraticCurveTo(-half, half, -half, half - radius);
        shape.lineTo(-half, -half + radius);
        shape.quadraticCurveTo(-half, -half, -half + radius, -half);
        const curve = new THREE.CatmullRomCurve3(shape.getPoints(16).map((point) => new THREE.Vector3(point.x, point.y, 0)), true);

        for (let index = 0; index < 6; index++) {
          const ring = new THREE.Group();
          ring.position.set((index - 2.5) * 0.11, (index - 2.5) * -0.07, (index - 2.5) * 0.3);
          ring.rotation.z = index * 0.07;
          const geometry = new THREE.TubeGeometry(curve, 192, index === 2 ? 0.012 : 0.006, 5, true);
          geometries.push(geometry);
          const colors = [];
          for (let point = 0; point < geometry.attributes.position.count; point++) {
            const segment = Math.floor(point / 6) / 192;
            const color = new THREE.Color(0xe5e5e5);
            if (index === 2) {
              if (segment < 0.17) color.lerp(new THREE.Color(0xffb931), 1 - segment / 0.17);
              if (segment > 0.48 && segment < 0.78) color.lerp(new THREE.Color(0x2e73fc), Math.sin((segment - 0.48) / 0.3 * Math.PI));
            }
            colors.push(color.r, color.g, color.b);
          }
          geometry.setAttribute("color", new THREE.Float32BufferAttribute(colors, 3));
          const material = new THREE.MeshBasicMaterial({ vertexColors: true, transparent: true, opacity: index === 2 ? 1 : 0.27 + index * 0.07 });
          materials.push(material);
          ring.add(new THREE.Mesh(geometry, material));
          if (index === 2) {
            const haloGeometry = new THREE.TubeGeometry(curve, 192, 0.055, 5, true);
            const haloMaterial = new THREE.MeshBasicMaterial({ color: 0xe1e9ff, transparent: true, opacity: 0.035, blending: THREE.AdditiveBlending, depthWrite: false });
            geometries.push(haloGeometry);
            materials.push(haloMaterial);
            ring.add(new THREE.Mesh(haloGeometry, haloMaterial));
          }
          lattice.add(ring);
          lines.push(ring);
        }

        let pointerX = 0;
        let pointerY = 0;
        let contextLost = false;
        let lastFrame = 0;
        let phase = 0;
        const draw = () => {
          lattice.rotation.set(0.4 + pointerY * 0.08, -0.3 + pointerX * 0.1, 0.6 + Math.sin(phase) * 0.045);
          lines.forEach((line, index) => { line.rotation.z = index * 0.07 + Math.sin(phase + index * 0.35) * 0.025; });
          renderer.render(scene, camera);
        };
        const frame = (time: number) => {
          // Cap the focal animation at 30fps; the rest of the page stays still.
          if (time - lastFrame < 32) return;
          phase += Math.min((time - lastFrame) / 1000, 0.1) * 0.18;
          lastFrame = time;
          draw();
        };
        const updatePlayback = () => {
          if (contextLost || cancelled) return;
          const animate = visible && !document.hidden && !paused.current;
          renderer.setAnimationLoop(animate ? frame : null);
          lastFrame = performance.now();
          if (visible) draw();
        };
        syncPlayback.current = updatePlayback;
        const resize = new ResizeObserver(() => {
          const { width, height } = element!.getBoundingClientRect();
          if (!width || !height) return;
          const aspect = width / height;
          camera.left = -3.3 * aspect;
          camera.right = 3.3 * aspect;
          camera.updateProjectionMatrix();
          renderer.setSize(width, height, false);
          if (!contextLost) draw();
        });
        resize.observe(element!);
        const pointerMove = (event: PointerEvent) => {
          if (event.pointerType !== "mouse" || paused.current) return;
          const bounds = element!.getBoundingClientRect();
          pointerX = (event.clientX - bounds.left) / bounds.width * 2 - 1;
          pointerY = (event.clientY - bounds.top) / bounds.height * 2 - 1;
        };
        const pointerLeave = () => { pointerX = 0; pointerY = 0; };
        const lost = (event: Event) => {
          event.preventDefault();
          contextLost = true;
          renderer.setAnimationLoop(null);
          setReady(false);
        };
        const restored = () => { contextLost = false; setReady(true); updatePlayback(); };
        element!.addEventListener("pointermove", pointerMove);
        element!.addEventListener("pointerleave", pointerLeave);
        document.addEventListener("visibilitychange", updatePlayback);
        renderer.domElement.addEventListener("webglcontextlost", lost);
        renderer.domElement.addEventListener("webglcontextrestored", restored);
        cleanupRenderer = () => {
          resize.disconnect();
          document.removeEventListener("visibilitychange", updatePlayback);
          element!.removeEventListener("pointermove", pointerMove);
          element!.removeEventListener("pointerleave", pointerLeave);
          renderer.domElement.removeEventListener("webglcontextlost", lost);
          renderer.domElement.removeEventListener("webglcontextrestored", restored);
          renderer.setAnimationLoop(null);
          geometries.forEach((geometry) => geometry.dispose());
          materials.forEach((material) => material.dispose());
          renderer.dispose();
          renderer.forceContextLoss();
          renderer.domElement.remove();
          syncPlayback.current = null;
        };
        setReady(true);
        updatePlayback();
      } catch {
        // Keep the static geometry when WebGL is unavailable or the import fails.
        cleanupRenderer();
      }
    }

    const observer = new IntersectionObserver(([entry]) => {
      visible = entry.isIntersecting;
      if (visible) void initialize();
      syncPlayback.current?.();
    }, { threshold: 0.01 });
    observer.observe(element);
    return () => { cancelled = true; observer.disconnect(); cleanupRenderer(); };
  }, []);

  return (
    <div className="hero-artwork">
      <div className="hero-lattice" ref={host} data-ready={ready}>
        <svg className="lattice-fallback" viewBox="0 0 600 500" fill="none" aria-hidden="true">
          {[0, 1, 2, 3, 4, 5].map((index) => <rect key={index} x="152" y="102" width="280" height="280" rx="22" stroke={index === 2 ? "#e5e5e5" : "#464646"} transform={`rotate(${30 + index * 5} 300 250) translate(${index * 5 - 15} ${index * 4 - 10})`} />)}
        </svg>
      </div>
      {ready && !reducedMotion && <button type="button" className="lattice-motion" onClick={() => setPlaying(!playing)} aria-label={playing ? "Pause graphic animation" : "Play graphic animation"}>
        {playing ? <Pause size={13} aria-hidden="true" /> : <Play size={13} aria-hidden="true" />}
      </button>}
      <div className="lattice-caption"><span aria-hidden="true" /> Build, deploy, and run on your server.</div>
    </div>
  );
}
