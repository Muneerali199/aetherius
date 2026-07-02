import { useEffect, useRef } from 'react';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';

gsap.registerPlugin(ScrollTrigger);

const capabilities = [
  {
    id: 'compute',
    label: '01',
    title: 'Elastic Compute',
    description:
      'Deploy workloads to a mesh of heterogeneous GPU and CPU nodes. Aetherius auto-scales across the global network, routing tasks to the nearest available compute resource with sub-millisecond latency.',
    image: '/images/server-racks.jpg',
  },
  {
    id: 'storage',
    label: '02',
    title: 'Distributed Storage',
    description:
      'Data is erasure-coded and sharded across geographically distributed nodes. Redundancy is automatic, retrieval is parallel, and integrity is cryptographically verified on every read.',
    image: '/images/data-viz.jpg',
  },
  {
    id: 'inference',
    label: '03',
    title: 'Inference at the Edge',
    description:
      'AI models run where the data lives. Our edge inference engine automatically partitions and distributes model weights, enabling real-time AI without centralized data centers.',
    image: '/images/chip.jpg',
  },
  {
    id: 'network',
    label: '04',
    title: 'Mesh Networking',
    description:
      'A self-healing, encrypted overlay network connects every node. Traffic routes around congestion automatically, maintaining optimal throughput even under partial network failure.',
    image: '/images/abyss.jpg',
  },
];

export default function Capabilities() {
  const sectionRef = useRef<HTMLElement>(null);
  const imagesRef = useRef<HTMLDivElement>(null);
  const itemRefs = useRef<(HTMLDivElement | null)[]>([]);

  useEffect(() => {
    const ctx = gsap.context(() => {
      // Animate each capability item on scroll
      itemRefs.current.forEach((item, i) => {
        if (!item) return;

        gsap.fromTo(
          item,
          { opacity: 0.25 },
          {
            opacity: 1,
            scrollTrigger: {
              trigger: item,
              start: 'top 70%',
              end: 'top 30%',
              scrub: true,
            },
          }
        );

        // Dim previous items
        if (i > 0) {
          gsap.to(itemRefs.current[i - 1], {
            opacity: 0.25,
            scrollTrigger: {
              trigger: item,
              start: 'top 70%',
              end: 'top 50%',
              scrub: true,
            },
          });
        }
      });

      // Parallax on images
      const imageEls = imagesRef.current?.querySelectorAll('.cap-image');
      imageEls?.forEach((img) => {
        gsap.fromTo(
          img,
          { y: 60, scale: 1.1 },
          {
            y: -60,
            scale: 1,
            scrollTrigger: {
              trigger: img,
              start: 'top bottom',
              end: 'bottom top',
              scrub: true,
            },
          }
        );
      });
    }, sectionRef);

    return () => ctx.revert();
  }, []);

  return (
    <section
      id="capabilities"
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
        {/* Section header */}
        <div className="mb-20">
          <div
            className="mb-4 font-mono text-xs tracking-[0.2em]"
            style={{ color: 'var(--core-blue)' }}
          >
            CORE CAPABILITIES
          </div>
          <h2
            className="max-w-[600px] font-sans text-[clamp(1.8rem,3.5vw,3rem)] font-normal leading-tight tracking-[-0.01em]"
            style={{ color: 'var(--surface-mist)' }}
          >
            Four layers of autonomous infrastructure.
          </h2>
        </div>

        {/* Two-column layout */}
        <div className="flex flex-col gap-16 lg:flex-row lg:gap-24">
          {/* Left: capability list */}
          <div className="flex-1">
            <div className="flex flex-col gap-16 lg:gap-24">
              {capabilities.map((cap, i) => (
                <div
                  key={cap.id}
                  ref={(el) => { itemRefs.current[i] = el; }}
                  className="transition-opacity duration-300"
                >
                  <div className="mb-4 flex items-center gap-4">
                    <span
                      className="font-mono text-xs"
                      style={{ color: 'var(--core-blue)' }}
                    >
                      {cap.label}
                    </span>
                    <div
                      className="h-px flex-1"
                      style={{
                        background:
                          'linear-gradient(to right, rgba(0, 119, 182, 0.3), transparent)',
                      }}
                    />
                  </div>
                  <h3
                    className="mb-4 font-sans text-2xl font-normal tracking-tight md:text-3xl"
                    style={{ color: 'var(--surface-mist)' }}
                  >
                    {cap.title}
                  </h3>
                  <p
                    className="max-w-[480px] text-sm leading-relaxed"
                    style={{ color: 'rgba(202, 240, 248, 0.5)' }}
                  >
                    {cap.description}
                  </p>
                </div>
              ))}
            </div>
          </div>

          {/* Right: sticky images */}
          <div className="hidden lg:block lg:w-[45%]">
            <div ref={imagesRef} className="sticky top-[15vh] flex flex-col gap-8">
              {capabilities.map((cap) => (
                <div
                  key={cap.id}
                  className="cap-image relative overflow-hidden rounded-sm"
                  style={{
                    aspectRatio: '4/3',
                    border: '1px solid rgba(0, 119, 182, 0.15)',
                  }}
                >
                  <img
                    src={cap.image}
                    alt={cap.title}
                    className="h-full w-full object-cover"
                    loading="lazy"
                  />
                  <div
                    className="absolute inset-0"
                    style={{
                      background:
                        'linear-gradient(to top, rgba(0, 8, 20, 0.6) 0%, transparent 50%)',
                    }}
                  />
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
