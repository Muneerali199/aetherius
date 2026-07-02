import FluidBackground from '@/components/effects/FluidBackground';
import Navigation from '@/components/Navigation';
import Hero from '@/sections/Hero';
import Wake from '@/sections/Wake';
import Capabilities from '@/sections/Capabilities';
import Manifesto from '@/sections/Manifesto';
import NetworkStats from '@/sections/NetworkStats';
import TerminalFooter from '@/sections/TerminalFooter';
import { useLenis } from '@/hooks/useLenis';

export default function Home() {
  useLenis();

  return (
    <div className="relative" style={{ backgroundColor: 'var(--abyssal-black)' }}>
      <FluidBackground />
      <Navigation />
      <main className="relative">
        <Hero />
        <Wake />
        <Capabilities />
        <Manifesto />
        <NetworkStats />
      </main>
      <TerminalFooter />
    </div>
  );
}
