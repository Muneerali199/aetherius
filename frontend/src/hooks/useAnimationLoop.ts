import { useEffect, useRef } from 'react';

const subscribers = new Set<(time: number) => void>();
let rafId: number | null = null;

function runLoop(time: number) {
  subscribers.forEach(cb => cb(time));
  rafId = requestAnimationFrame(runLoop);
}

export function useAnimationLoop(callback: (time: number) => void) {
  const callbackRef = useRef(callback);
  callbackRef.current = callback;

  useEffect(() => {
    const wrapper = (time: number) => callbackRef.current(time);
    subscribers.add(wrapper);
    if (subscribers.size === 1) {
      rafId = requestAnimationFrame(runLoop);
    }
    return () => {
      subscribers.delete(wrapper);
      if (subscribers.size === 0 && rafId !== null) {
        cancelAnimationFrame(rafId);
        rafId = null;
      }
    };
  }, []);
}
