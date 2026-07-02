import { useEffect, useRef } from 'react';

const footerLinks = [
  {
    heading: 'Platform',
    links: ['Compute', 'Storage', 'Networking', 'Inference', 'Pricing'],
  },
  {
    heading: 'Developers',
    links: ['Documentation', 'API Reference', 'SDKs', 'CLI Tools', 'Status'],
  },
  {
    heading: 'Network',
    links: ['Node Map', 'Explorer', 'Governance', 'Staking', 'Tokenomics'],
  },
  {
    heading: 'Company',
    links: ['About', 'Blog', 'Careers', 'Brand', 'Contact'],
  },
];

export default function TerminalFooter() {
  const path1Ref = useRef<SVGTextPathElement>(null);
  const path2Ref = useRef<SVGTextPathElement>(null);
  const rafRef = useRef<number>(0);

  useEffect(() => {
    let offset1 = 0;
    let offset2 = 980;
    const speed = 0.15;

    const animate = () => {
      rafRef.current = requestAnimationFrame(animate);

      offset1 -= speed;
      offset2 -= speed;

      if (offset1 <= -980) offset1 = 0;
      if (offset2 <= 0) offset2 = 980;

      if (path1Ref.current) {
        path1Ref.current.setAttribute('startOffset', String(offset1));
      }
      if (path2Ref.current) {
        path2Ref.current.setAttribute('startOffset', String(offset2));
      }
    };

    rafRef.current = requestAnimationFrame(animate);

    return () => cancelAnimationFrame(rafRef.current);
  }, []);

  return (
    <footer
      id="docs"
      className="relative"
      style={{
        zIndex: 2,
        backgroundColor: 'var(--abyssal-black)',
        borderTop: '1px solid rgba(0, 119, 182, 0.12)',
      }}
    >
      {/* SVG text loop banner */}
      <div
        className="overflow-hidden py-8"
        style={{ borderBottom: '1px solid rgba(0, 119, 182, 0.08)' }}
      >
        <svg
          viewBox="0 0 240 24"
          className="w-full"
          style={{ minWidth: '1200px' }}
          preserveAspectRatio="xMidYMid meet"
        >
          <defs>
            <path
              id="textPathCurve"
              d="M0,12 Q60,-12 120,12 T240,12"
              fill="none"
            />
          </defs>
          <text
            style={{
              fontFamily: 'JetBrains Mono, monospace',
              fontSize: '8px',
              fill: 'rgba(0, 119, 182, 0.35)',
              letterSpacing: '0.08em',
            }}
          >
            <textPath
              ref={path1Ref}
              href="#textPathCurve"
              startOffset="0"
            >
              DISTRIBUTED COMPUTE NETWORK — AUTONOMOUS INFRASTRUCTURE — GLOBAL MESH — EDGE INFERENCE — DECENTRALIZED STORAGE —
            </textPath>
            <textPath
              ref={path2Ref}
              href="#textPathCurve"
              startOffset="980"
            >
              DISTRIBUTED COMPUTE NETWORK — AUTONOMOUS INFRASTRUCTURE — GLOBAL MESH — EDGE INFERENCE — DECENTRALIZED STORAGE —
            </textPath>
          </text>
        </svg>
      </div>

      {/* Footer columns */}
      <div className="mx-auto max-w-[1400px] px-6 py-16 md:px-12">
        <div className="grid grid-cols-2 gap-12 md:grid-cols-4">
          {footerLinks.map((col) => (
            <div key={col.heading}>
              <div
                className="mb-6 font-mono text-[10px] tracking-[0.15em]"
                style={{ color: 'var(--core-blue)' }}
              >
                {col.heading}
              </div>
              <ul className="flex flex-col gap-3">
                {col.links.map((link) => (
                  <li key={link}>
                    <a
                      href="#"
                      className="text-sm transition-colors duration-300 hover:text-mist"
                      style={{ color: 'rgba(202, 240, 248, 0.45)' }}
                    >
                      {link}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* Bottom bar */}
        <div
          className="mt-16 flex flex-col items-center justify-between gap-4 border-t pt-8 md:flex-row"
          style={{ borderColor: 'rgba(0, 119, 182, 0.1)' }}
        >
          <div className="flex items-center gap-2">
            <div
              className="flex h-6 w-6 items-center justify-center rounded-full"
              style={{ border: '1px solid rgba(144, 224, 239, 0.3)' }}
            >
              <div
                className="h-1.5 w-1.5 rounded-full"
                style={{ backgroundColor: 'var(--glacial-cyan)' }}
              />
            </div>
            <span
              className="font-mono text-[10px] tracking-[0.15em]"
              style={{ color: 'rgba(202, 240, 248, 0.35)' }}
            >
              AETHERIUS
            </span>
          </div>

          <p
            className="font-mono text-[10px] tracking-wider"
            style={{ color: 'rgba(202, 240, 248, 0.25)' }}
          >
            © 2026 Aetherius Network. All systems autonomous.
          </p>
        </div>
      </div>
    </footer>
  );
}
