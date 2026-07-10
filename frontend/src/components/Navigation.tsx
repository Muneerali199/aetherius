import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router';

const navLinks = [
  { label: 'Marketplace', href: '/marketplace' },
  { label: 'Infrastructure', href: '#capabilities' },
  { label: 'Manifesto', href: '#manifesto' },
  { label: 'Network', href: '#network' },
  { label: 'Documentation', href: '#docs' },
];

export default function Navigation() {
  const [scrolled, setScrolled] = useState(false);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const navRef = useRef<HTMLElement>(null);
  const navigate = useNavigate();

  useEffect(() => {
    setIsLoggedIn(!!localStorage.getItem('access_token'));
  }, []);

  useEffect(() => {
    const onScroll = () => {
      setScrolled(window.scrollY > window.innerHeight * 0.8);
    };
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  }, []);

  return (
    <nav
      ref={navRef}
      className="fixed top-0 left-0 right-0 z-50 transition-all duration-700"
      style={{
        backgroundColor: scrolled ? 'rgba(0, 13, 29, 0.85)' : 'transparent',
        backdropFilter: scrolled ? 'blur(20px)' : 'none',
        borderBottom: scrolled ? '1px solid rgba(0, 119, 182, 0.15)' : '1px solid transparent',
      }}
    >
      <div className="mx-auto flex max-w-[1400px] items-center justify-between px-6 py-4 md:px-12">
        <a href="#" className="flex items-center gap-2">
          <div
            className="flex h-8 w-8 items-center justify-center rounded-full"
            style={{ border: '1px solid rgba(144, 224, 239, 0.4)' }}
          >
            <div
              className="h-2 w-2 rounded-full"
              style={{ backgroundColor: 'var(--glacial-cyan)' }}
            />
          </div>
          <span
            className="font-mono text-sm tracking-[0.15em]"
            style={{ color: 'var(--surface-mist)' }}
          >
            AETHERIUS
          </span>
        </a>

        <div className="hidden items-center gap-8 md:flex">
          {navLinks.map((link) => (
            <a
              key={link.label}
              href={link.href}
              className="font-mono-data transition-colors duration-300 hover:text-mist"
              style={{ color: 'rgba(202, 240, 248, 0.6)' }}
              onClick={(e) => {
                if (link.href.startsWith('/')) {
                  // Internal route — let browser navigate
                  return;
                }
                e.preventDefault();
                const el = document.querySelector(link.href);
                if (el) el.scrollIntoView({ behavior: 'smooth' });
              }}
            >
              {link.label}
            </a>
          ))}
        </div>

        <div className="hidden items-center gap-3 md:flex">
          {isLoggedIn ? (
            <a
              href="/dashboard"
              className="flex items-center gap-2 rounded-full px-5 py-2 font-mono text-xs tracking-wider transition-all duration-300 hover:brightness-110"
              style={{
                border: '1px solid rgba(0, 119, 182, 0.4)',
                color: 'var(--glacial-cyan)',
              }}
              onClick={(e) => { e.preventDefault(); navigate('/dashboard'); }}
            >
              Dashboard
            </a>
          ) : (
            <a
              href="/login"
              className="rounded-full px-5 py-2 font-mono text-xs tracking-wider transition-all duration-300 hover:brightness-110"
              style={{
                border: '1px solid rgba(0, 119, 182, 0.3)',
                color: 'rgba(202, 240, 248, 0.7)',
              }}
              onClick={(e) => { e.preventDefault(); navigate('/login'); }}
            >
              Sign In
            </a>
          )}
          <div
            className="flex items-center gap-2 rounded-full px-5 py-2 font-mono text-xs tracking-wider"
            style={{
              border: '1px solid rgba(0, 119, 182, 0.4)',
              color: 'var(--glacial-cyan)',
            }}
          >
            <span className="h-1.5 w-1.5 rounded-full bg-green-400 animate-pulse" />
            Network Live
          </div>
        </div>
      </div>
    </nav>
  );
}
