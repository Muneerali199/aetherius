import { useEffect, useRef } from 'react';

const statusPills = [
  { label: 'GPU Core', value: '98%' },
  { label: 'Render', value: 'Complete' },
  { label: 'Latency', value: '0.4ms' },
  { label: 'Nodes', value: '12,847' },
];

export default function Wake() {
  const sectionRef = useRef<HTMLElement>(null);
  const videoRef = useRef<HTMLVideoElement>(null);

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
      { threshold: 0.3 }
    );

    if (sectionRef.current) observer.observe(sectionRef.current);
    return () => observer.disconnect();
  }, []);

  return (
    <section
      ref={sectionRef}
      className="relative w-full overflow-hidden"
      style={{ height: '70vh', zIndex: 2 }}
    >
      <video
        ref={videoRef}
        className="absolute inset-0 h-full w-full object-cover"
        src="/videos/architecture.mp4"
        muted
        loop
        playsInline
        preload="metadata"
      />

      {/* Dark overlay for depth */}
      <div
        className="absolute inset-0"
        style={{
          background: 'linear-gradient(to bottom, rgba(0, 8, 20, 0.3) 0%, rgba(0, 8, 20, 0.1) 40%, rgba(0, 8, 20, 0.3) 100%)',
        }}
      />

      {/* Status pills */}
      <div className="absolute bottom-8 left-1/2 flex -translate-x-1/2 gap-3">
        {statusPills.map((pill) => (
          <div
            key={pill.label}
            className="flex items-center gap-2 rounded-full px-4 py-2 font-mono text-[10px] tracking-wider"
            style={{
              background: 'rgba(0, 8, 20, 0.7)',
              border: '1px solid rgba(0, 119, 182, 0.25)',
              color: 'var(--glacial-cyan)',
              backdropFilter: 'blur(8px)',
            }}
          >
            <span
              className="h-1.5 w-1.5 rounded-full"
              style={{
                backgroundColor:
                  pill.label === 'GPU Core'
                    ? '#4ade80'
                    : pill.label === 'Render'
                    ? '#90E0EF'
                    : pill.label === 'Latency'
                    ? '#fbbf24'
                    : '#60a5fa',
                boxShadow: `0 0 6px ${
                  pill.label === 'GPU Core'
                    ? '#4ade80'
                    : pill.label === 'Render'
                    ? '#90E0EF'
                    : pill.label === 'Latency'
                    ? '#fbbf24'
                    : '#60a5fa'
                }`,
              }}
            />
            {pill.label}: {pill.value}
          </div>
        ))}
      </div>
    </section>
  );
}
