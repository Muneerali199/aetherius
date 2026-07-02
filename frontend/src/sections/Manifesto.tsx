import { useEffect, useRef } from 'react';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';

gsap.registerPlugin(ScrollTrigger);

const manifestoLines = [
  'The future of compute is not centralized.',
  'It is not owned by corporations.',
  'It is distributed, like consciousness.',
  'Every device a node. Every connection a synapse.',
  'Aetherius is the nervous system of a new internet.',
  'Autonomous. Encrypted. Unstoppable.',
];

export default function Manifesto() {
  const sectionRef = useRef<HTMLElement>(null);
  const videoRef = useRef<HTMLVideoElement>(null);
  const linesRef = useRef<(HTMLDivElement | null)[]>([]);

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting && videoRef.current) {
            videoRef.current.play();
          } else if (videoRef.current) {
            videoRef.current.pause();
          }
        });
      },
      { threshold: 0.2 }
    );

    if (sectionRef.current) observer.observe(sectionRef.current);

    const ctx = gsap.context(() => {
      linesRef.current.forEach((line, i) => {
        if (!line) return;
        gsap.fromTo(
          line,
          {
            opacity: 0,
            y: 40,
          },
          {
            opacity: 1,
            y: 0,
            duration: 1,
            ease: 'power3.out',
            scrollTrigger: {
              trigger: line,
              start: 'top 80%',
              end: 'top 50%',
              toggleActions: 'play none none reverse',
            },
            delay: i * 0.1,
          }
        );
      });
    }, sectionRef);

    return () => {
      observer.disconnect();
      ctx.revert();
    };
  }, []);

  return (
    <section
      id="manifesto"
      ref={sectionRef}
      className="relative overflow-hidden"
      style={{
        zIndex: 2,
        minHeight: '100vh',
        paddingTop: '20vh',
        paddingBottom: '20vh',
      }}
    >
      {/* Background video */}
      <video
        ref={videoRef}
        className="absolute inset-0 h-full w-full object-cover"
        src="/videos/dataflow.mp4"
        muted
        loop
        playsInline
        preload="metadata"
        style={{ filter: 'grayscale(0.3) brightness(0.4)' }}
      />

      {/* Dark overlay */}
      <div
        className="absolute inset-0"
        style={{
          background:
            'linear-gradient(to bottom, var(--abyssal-black) 0%, rgba(0, 8, 20, 0.7) 20%, rgba(0, 8, 20, 0.7) 80%, var(--abyssal-black) 100%)',
        }}
      />

      <div className="relative mx-auto max-w-[900px] px-6 md:px-12">
        <div
          className="mb-16 font-mono text-xs tracking-[0.2em]"
          style={{ color: 'var(--core-blue)' }}
        >
          MANIFESTO
        </div>

        <div className="flex flex-col gap-8">
          {manifestoLines.map((line, i) => (
            <div
              key={i}
              ref={(el) => { linesRef.current[i] = el; }}
            >
              <p
                className="font-sans text-[clamp(1.5rem,3vw,2.5rem)] font-light leading-tight tracking-[-0.01em]"
                style={{ color: 'var(--surface-mist)' }}
              >
                {line}
              </p>
            </div>
          ))}
        </div>

        <div
          className="mt-16 h-px w-24"
          style={{
            background:
              'linear-gradient(to right, var(--core-blue), transparent)',
          }}
        />

        <p
          className="mt-8 max-w-[500px] text-sm leading-relaxed"
          style={{ color: 'rgba(202, 240, 248, 0.45)' }}
        >
          We are building the substrate for a world where compute is as
          accessible as sunlight. Where intelligence flows through the mesh
          like water finding its level. This is Aetherius.
        </p>
      </div>
    </section>
  );
}
