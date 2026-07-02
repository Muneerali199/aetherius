import { useEffect, useRef, useState } from 'react';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';

gsap.registerPlugin(ScrollTrigger);

const stats = [
  { label: 'Active Nodes', value: 12847, suffix: '' },
  { label: 'Countries', value: 142, suffix: '' },
  { label: 'GPU Hours / Day', value: 2.4, suffix: 'M', isDecimal: true },
  { label: 'Avg Latency', value: 0.4, suffix: 'ms', isDecimal: true },
];

function AnimatedCounter({
  value,
  suffix,
  isDecimal,
  inView,
}: {
  value: number;
  suffix: string;
  isDecimal?: boolean;
  inView: boolean;
}) {
  const [display, setDisplay] = useState('0');
  const ref = useRef({ current: 0 });

  useEffect(() => {
    if (!inView) return;

    const obj = { val: 0 };
    const tween = gsap.to(obj, {
      val: value,
      duration: 2,
      ease: 'power2.out',
      onUpdate: () => {
        ref.current.current = obj.val;
        if (isDecimal) {
          setDisplay(obj.val.toFixed(1));
        } else {
          setDisplay(Math.floor(obj.val).toLocaleString());
        }
      },
    });

    return () => {
      tween.kill();
    };
  }, [inView, value, isDecimal]);

  return (
    <span>
      {display}
      {suffix}
    </span>
  );
}

export default function NetworkStats() {
  const sectionRef = useRef<HTMLElement>(null);
  const [inView, setInView] = useState(false);

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setInView(true);
        }
      },
      { threshold: 0.4 }
    );

    if (sectionRef.current) observer.observe(sectionRef.current);
    return () => observer.disconnect();
  }, []);

  return (
    <section
      id="network"
      ref={sectionRef}
      className="relative"
      style={{
        zIndex: 2,
        backgroundColor: 'var(--abyssal-black)',
        paddingTop: '15vh',
        paddingBottom: '15vh',
      }}
    >
      <div className="mx-auto max-w-[1400px] px-6 md:px-12">
        <div className="mb-16 text-center">
          <div
            className="mb-4 font-mono text-xs tracking-[0.2em]"
            style={{ color: 'var(--core-blue)' }}
          >
            LIVE NETWORK
          </div>
          <h2
            className="font-sans text-[clamp(1.5rem,3vw,2.5rem)] font-normal leading-tight"
            style={{ color: 'var(--surface-mist)' }}
          >
            The mesh is growing in real time.
          </h2>
        </div>

        <div className="grid grid-cols-2 gap-8 md:grid-cols-4">
          {stats.map((stat) => (
            <div
              key={stat.label}
              className="flex flex-col items-center gap-3 rounded-lg p-6"
              style={{
                border: '1px solid rgba(0, 119, 182, 0.12)',
                background: 'rgba(0, 29, 61, 0.3)',
              }}
            >
              <div
                className="font-sans text-[clamp(2rem,4vw,3.5rem)] font-light tracking-tight"
                style={{ color: 'var(--glacial-cyan)' }}
              >
                <AnimatedCounter
                  value={stat.value}
                  suffix={stat.suffix}
                  isDecimal={stat.isDecimal}
                  inView={inView}
                />
              </div>
              <div
                className="font-mono text-[10px] tracking-[0.15em]"
                style={{ color: 'rgba(202, 240, 248, 0.4)' }}
              >
                {stat.label}
              </div>
            </div>
          ))}
        </div>

        {/* Server video strip */}
        <div className="mt-16 overflow-hidden rounded-sm">
          <video
            className="h-auto w-full object-cover"
            style={{ maxHeight: '400px' }}
            src="/videos/servers.mp4"
            muted
            loop
            playsInline
            autoPlay
            preload="metadata"
          />
        </div>
      </div>
    </section>
  );
}
