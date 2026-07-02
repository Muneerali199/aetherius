import { useEffect, useRef, useState } from 'react';

interface LogEntry {
  id: string;
  label: string;
  value: string;
  type: 'metric' | 'status' | 'hash';
}

function generateLog(): LogEntry {
  const types: LogEntry['type'][] = ['metric', 'status', 'hash'];
  const type = types[Math.floor(Math.random() * types.length)];
  const nodes = ['TX_01', 'TX_02', 'TX_03', 'TX_04', 'SEC_AUTH', 'GPU_01', 'GPU_02', 'NODE_EU', 'NODE_AS', 'NODE_US'];
  const node = nodes[Math.floor(Math.random() * nodes.length)];

  if (type === 'metric') {
    return { id: Math.random().toString(36), label: node, value: `${Math.floor(Math.random() * 20 + 1)}ms`, type };
  } else if (type === 'status') {
    const statuses = ['OK', 'SYNC', 'ACTIVE', 'READY'];
    return { id: Math.random().toString(36), label: node, value: statuses[Math.floor(Math.random() * statuses.length)], type };
  } else {
    const hex = Array.from({ length: 4 }, () => Math.floor(Math.random() * 16).toString(16).toUpperCase()).join('');
    return { id: Math.random().toString(36), label: node, value: `0x${hex}`, type };
  }
}

export default function Hero() {
  const [logs, setLogs] = useState<LogEntry[][]>([
    Array.from({ length: 5 }, generateLog),
    Array.from({ length: 5 }, generateLog),
    Array.from({ length: 5 }, generateLog),
    Array.from({ length: 5 }, generateLog),
  ]);
  const intervalRef = useRef<ReturnType<typeof setInterval>>(null);

  useEffect(() => {
    intervalRef.current = setInterval(() => {
      setLogs((prev) => {
        const colIndex = Math.floor(Math.random() * 4);
        const newLogs = prev.map((col, i) => {
          if (i !== colIndex) return col;
          const newCol = [...col];
          newCol.shift();
          newCol.push(generateLog());
          return newCol;
        });
        return newLogs;
      });
    }, 800);

    return () => {
      if (intervalRef.current !== null) clearInterval(intervalRef.current);
    };
  }, []);

  return (
    <section
      className="relative flex min-h-screen flex-col items-center justify-center px-6"
      style={{ zIndex: 1 }}
    >
      <div className="relative z-10 mx-auto max-w-[1200px] text-center">
        <div
          className="pointer-events-none absolute inset-0 -translate-y-12"
          style={{
            background:
              'radial-gradient(ellipse at center, rgba(0,8,20,0.6) 0%, transparent 70%)',
          }}
        />
        <div
          className="mb-6 inline-block font-mono text-xs tracking-[0.2em]"
          style={{ color: 'var(--core-blue)' }}
        >
          DISTRIBUTED COMPUTE NETWORK
        </div>

        <h1
          className="mb-8 font-sans text-[clamp(2.5rem,8vw,7rem)] font-semibold leading-[0.9] tracking-[-0.02em]"
          style={{ color: '#FFFFFF' }}
        >
          Autonomous
          <br />
          Infrastructure.
        </h1>

        <p
          className="mx-auto mb-12 max-w-[540px] text-base leading-relaxed"
          style={{ color: 'rgba(255, 255, 255, 0.75)' }}
        >
          Aetherius orchestrates idle compute resources across the globe into a
          single, self-managing infrastructure layer. No data centers. No
          boundaries. Pure computational fluidity.
        </p>

        <div className="flex items-center justify-center gap-4">
          <a
            href="#capabilities"
            className="rounded-full px-8 py-3 font-mono text-xs tracking-wider transition-all duration-300 hover:scale-105"
            style={{
              backgroundColor: 'var(--core-blue)',
              color: 'var(--surface-mist)',
            }}
            onClick={(e) => {
              e.preventDefault();
              document.querySelector('#capabilities')?.scrollIntoView({ behavior: 'smooth' });
            }}
          >
            Explore the Network
          </a>
          <a
            href="#"
            className="rounded-full px-8 py-3 font-mono text-xs tracking-wider transition-all duration-300 hover:border-opacity-60"
            style={{
              border: '1px solid rgba(144, 224, 239, 0.3)',
              color: 'var(--glacial-cyan)',
            }}
          >
            Read Manifesto
          </a>
        </div>
      </div>

      {/* Live Node Status Grid */}
      <div
        className="absolute bottom-12 left-1/2 w-full max-w-[900px] -translate-x-1/2 px-6"
      >
        <div
          className="flex justify-between gap-2 overflow-hidden rounded-lg p-4 md:gap-4"
          style={{
            background: 'rgba(0, 8, 20, 0.6)',
            border: '1px solid rgba(0, 119, 182, 0.15)',
            backdropFilter: 'blur(12px)',
          }}
        >
          {logs.map((col, colIdx) => (
            <div key={colIdx} className="flex min-w-0 flex-1 flex-col gap-1.5">
              <div
                className="mb-2 font-mono text-[10px] tracking-[0.15em]"
                style={{ color: 'rgba(0, 119, 182, 0.7)' }}
              >
                NODE_0{colIdx + 1}
              </div>
              {col.map((log) => (
                <div
                  key={log.id}
                  className="flex items-center justify-between gap-2 font-mono text-[10px]"
                >
                  <span style={{ color: 'rgba(202, 240, 248, 0.4)' }}>
                    {log.label}
                  </span>
                  <span
                    style={{
                      color:
                        log.type === 'status'
                          ? 'rgba(74, 222, 128, 0.8)'
                          : log.type === 'hash'
                          ? 'rgba(144, 224, 239, 0.7)'
                          : 'rgba(202, 240, 248, 0.6)',
                    }}
                  >
                    {log.value}
                  </span>
                </div>
              ))}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
