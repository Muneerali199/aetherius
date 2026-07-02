import { useEffect, useRef } from 'react';
import * as THREE from 'three';
import { useAnimationLoop } from '@/hooks/useAnimationLoop';

const vertexShader = `
varying vec2 vUv;
void main() {
  vUv = uv;
  gl_Position = vec4(position, 1.0);
}
`;

const fragmentShader = `
precision highp float;
varying vec2 vUv;

uniform float u_time;
uniform vec2 u_mouse;
uniform vec2 u_resolution;
uniform float u_rotationSpeed;
uniform float u_warpIntensity;
uniform float u_colorShift;

vec3 mod289v3(vec3 x) { return x - floor(x * (1.0 / 289.0)) * 289.0; }
vec4 mod289v4(vec4 x) { return x - floor(x * (1.0 / 289.0)) * 289.0; }
vec4 permute(vec4 x) { return mod289v4(((x * 34.0) + 1.0) * x); }
vec4 taylorInvSqrt(vec4 r) { return 1.79284291400159 - 0.85373472095314 * r; }

float snoise(vec3 v) {
  const vec2 C = vec2(1.0 / 6.0, 1.0 / 3.0);
  const vec4 D = vec4(0.0, 0.5, 1.0, 2.0);

  vec3 i = floor(v + dot(v, C.yyy));
  vec3 x0 = v - i + dot(i, C.xxx);

  vec3 g = step(x0.yzx, x0.xyz);
  vec3 l = 1.0 - g;
  vec3 i1 = min(g.xyz, l.zxy);
  vec3 i2 = max(g.xyz, l.zxy);

  vec3 x1 = x0 - i1 + C.xxx;
  vec3 x2 = x0 - i2 + C.yyy;
  vec3 x3 = x0 - D.yyy;

  i = mod289v3(i);
  vec4 p = permute(permute(permute(
    i.z + vec4(0.0, i1.z, i2.z, 1.0))
    + i.y + vec4(0.0, i1.y, i2.y, 1.0))
    + i.x + vec4(0.0, i1.x, i2.x, 1.0));

  float n_ = 0.142857142857;
  vec3 ns = n_ * D.wyz - D.xzx;

  vec4 j = p - 49.0 * floor(p * ns.z * ns.z);

  vec4 x_ = floor(j * ns.z);
  vec4 y_ = floor(j - 7.0 * x_);

  vec4 x = x_ * ns.x + ns.yyyy;
  vec4 y = y_ * ns.x + ns.yyyy;
  vec4 h = 1.0 - abs(x) - abs(y);

  vec4 b0 = vec4(x.xy, y.xy);
  vec4 b1 = vec4(x.zw, y.zw);

  vec4 s0 = floor(b0) * 2.0 + 1.0;
  vec4 s1 = floor(b1) * 2.0 + 1.0;
  vec4 sh = -step(h, vec4(0.0));

  vec4 a0 = b0.xzyw + s0.xzyw * sh.xxyy;
  vec4 a1 = b1.xzyw + s1.xzyw * sh.zzww;

  vec3 p0 = vec3(a0.xy, h.x);
  vec3 p1 = vec3(a0.zw, h.y);
  vec3 p2 = vec3(a1.xy, h.z);
  vec3 p3 = vec3(a1.zw, h.w);

  vec4 norm = taylorInvSqrt(vec4(dot(p0, p0), dot(p1, p1), dot(p2, p2), dot(p3, p3)));
  p0 *= norm.x;
  p1 *= norm.y;
  p2 *= norm.z;
  p3 *= norm.w;

  vec4 m = max(0.6 - vec4(dot(x0, x0), dot(x1, x1), dot(x2, x2), dot(x3, x3)), 0.0);
  m = m * m;
  return 42.0 * dot(m * m, vec4(dot(p0, x0), dot(p1, x1), dot(p2, x2), dot(p3, x3)));
}

float fbm(vec3 p, float rotationAngle) {
  float value = 0.0;
  float amplitude = 0.5;
  float frequency = 1.0;
  mat2 rot = mat2(
    cos(rotationAngle), -sin(rotationAngle),
    sin(rotationAngle), cos(rotationAngle)
  );

  for (int i = 0; i < 3; i++) {
    value += amplitude * snoise(p * frequency);
    p.xy = rot * p.xy;
    p *= 1.8;
    amplitude *= 0.5;
    frequency *= 2.2;
  }
  return value;
}

float sampleFluidState(vec2 coord, float t, float warp) {
  float aspect = u_resolution.x / u_resolution.y;
  coord *= vec2(aspect, 1.0);

  vec2 q = vec2(0.0);
  q.x = fbm(vec3(coord + vec2(1.7, 9.2) + 0.15 * t, 2.0), u_rotationSpeed);
  q.y = fbm(vec3(coord + vec2(8.3, 2.8) - 0.126 * t, 3.0), u_rotationSpeed);
  q *= warp;

  vec2 r = vec2(0.0);
  r.x = fbm(vec3(coord + 2.0 * q + vec2(1.7, 9.2) + 0.3 * t, 0.5), u_rotationSpeed);
  r.y = fbm(vec3(coord + 2.0 * q + vec2(8.3, 2.8) + 0.25 * t, 1.5), u_rotationSpeed);
  r *= warp;

  return fbm(vec3(coord + 2.0 * r, 6.0), u_rotationSpeed);
}

vec3 fluidColor(float fluidState) {
  vec3 color1 = vec3(0.0, 0.03, 0.08);
  vec3 color2 = vec3(0.0, 0.47, 0.71);
  vec3 color3 = vec3(0.56, 0.87, 0.93);

  float fractState = fract(fluidState * 2.0);
  if (fractState < 0.5) {
    return mix(color1, color2, fractState * 2.0);
  } else {
    return mix(color2, color3, (fractState - 0.5) * 2.0);
  }
}

void main() {
  float t = u_time * 0.2;

  float mouseInfluence = 1.0 - smoothstep(0.0, 0.5, distance(vUv, u_mouse));

  float fluidState = sampleFluidState(vUv, t, u_warpIntensity) + mouseInfluence * 0.2;

  vec3 finalColor = fluidColor(fluidState - u_colorShift);

  float highlight = smoothstep(0.6, 1.0, fluidState);
  finalColor += vec3(0.79, 0.94, 0.97) * highlight * 0.5;

  float darkMatter = smoothstep(0.1, 0.4, fluidState);
  finalColor -= vec3(0.0, 0.02, 0.04) * (1.0 - darkMatter);

  float coreGlow = smoothstep(0.3, 0.8, fluidState) * 0.3;
  finalColor += vec3(0.56, 0.87, 0.93) * coreGlow;

  finalColor = clamp(finalColor, 0.0, 1.0);
  finalColor = pow(finalColor, vec3(1.0 / 2.2));

  gl_FragColor = vec4(finalColor, 1.0);
}
`;

export default function FluidBackground() {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const mouseRef = useRef({ x: 0.5, y: 0.5 });
  const targetMouseRef = useRef({ x: 0.5, y: 0.5 });
  const startTimeRef = useRef(0);
  const frameCountRef = useRef(0);
  const rendererRef = useRef<THREE.WebGLRenderer | null>(null);
  const sceneRef = useRef<THREE.Scene | null>(null);
  const cameraRef = useRef<THREE.OrthographicCamera | null>(null);
  const uniformsRef = useRef<Record<string, { value: any }> | null>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const renderer = new THREE.WebGLRenderer({ canvas, antialias: false });
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 1.5));
    renderer.setSize(window.innerWidth, window.innerHeight);

    const scene = new THREE.Scene();
    const camera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0, 1);

    const uniforms = {
      u_time: { value: 0 },
      u_mouse: { value: new THREE.Vector2(0.5, 0.5) },
      u_resolution: { value: new THREE.Vector2(window.innerWidth, window.innerHeight) },
      u_rotationSpeed: { value: 0.2 },
      u_warpIntensity: { value: 0.8 },
      u_colorShift: { value: 0.1 },
    };

    const material = new THREE.ShaderMaterial({
      vertexShader,
      fragmentShader,
      uniforms,
    });

    const geometry = new THREE.PlaneGeometry(2, 2);
    const mesh = new THREE.Mesh(geometry, material);
    scene.add(mesh);

    rendererRef.current = renderer;
    sceneRef.current = scene;
    cameraRef.current = camera;
    uniformsRef.current = uniforms;

    const onMouseMove = (e: MouseEvent) => {
      targetMouseRef.current.x = e.clientX / window.innerWidth;
      targetMouseRef.current.y = 1.0 - e.clientY / window.innerHeight;
    };
    window.addEventListener('mousemove', onMouseMove);

    const onResize = () => {
      renderer.setSize(window.innerWidth, window.innerHeight);
      uniforms.u_resolution.value.set(window.innerWidth, window.innerHeight);
    };
    window.addEventListener('resize', onResize);

    startTimeRef.current = performance.now();

    return () => {
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('resize', onResize);
      geometry.dispose();
      material.dispose();
      renderer.dispose();
      rendererRef.current = null;
      sceneRef.current = null;
      cameraRef.current = null;
      uniformsRef.current = null;
    };
  }, []);

  useAnimationLoop(() => {
    frameCountRef.current++;
    if (frameCountRef.current % 2 !== 0) return;

    const renderer = rendererRef.current;
    const scene = sceneRef.current;
    const camera = cameraRef.current;
    const uniforms = uniformsRef.current;
    if (!renderer || !scene || !camera || !uniforms) return;

    const elapsed = (performance.now() - startTimeRef.current) / 1000;
    uniforms.u_time.value = elapsed;

    mouseRef.current.x += (targetMouseRef.current.x - mouseRef.current.x) * 0.05;
    mouseRef.current.y += (targetMouseRef.current.y - mouseRef.current.y) * 0.05;
    uniforms.u_mouse.value.set(mouseRef.current.x, mouseRef.current.y);

    renderer.render(scene, camera);
  });

  return (
    <canvas
      ref={canvasRef}
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        width: '100%',
        height: '100%',
        zIndex: 0,
      }}
    />
  );
}
