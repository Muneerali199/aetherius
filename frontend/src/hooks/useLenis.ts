import { useEffect, useRef } from 'react';
import Lenis from '@studio-freight/lenis';
import { useAnimationLoop } from './useAnimationLoop';

export function useLenis() {
  const lenisRef = useRef<Lenis | null>(null);

  useEffect(() => {
    const lenis = new Lenis({
      duration: 0.9,
      easing: (t: number) => Math.min(1, 1.001 - Math.pow(2, -10 * t)),
      orientation: 'vertical',
      gestureOrientation: 'vertical',
      smoothWheel: true,
      touchMultiplier: 2,
    });

    lenisRef.current = lenis;

    return () => {
      lenis.destroy();
    };
  }, []);

  useAnimationLoop((time) => {
    lenisRef.current?.raf(time);
  });

  return lenisRef;
}
