import { AnomalyFeed } from "@/components/AnomalyFeed";
import { ServiceHealthGrid } from "@/components/ServiceHealthGrid";
import { QueryTerminal } from "@/components/QueryTerminal";

export default function Home() {
  return (
    <main className="min-h-screen bg-bg p-6">
      <header className="mb-6 flex items-baseline justify-between">
        <div>
          <h1 className="font-mono text-lg font-semibold text-text">
            Sentinel<span className="text-accent">Ops</span>
          </h1>
          <p className="mt-0.5 text-sm text-muted">
            Distributed log intelligence and anomaly detection
          </p>
        </div>
      </header>

      <section className="mb-6">
        <ServiceHealthGrid />
      </section>

      <section className="grid grid-cols-1 gap-6 lg:grid-cols-2" style={{ height: "560px" }}>
        <AnomalyFeed />
        <QueryTerminal />
      </section>
    </main>
  );
}