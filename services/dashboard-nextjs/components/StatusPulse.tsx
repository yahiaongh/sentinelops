export function StatusPulse({ status }: { status: "healthy" | "warning" | "critical" }) {
  const colorClass = {
    healthy: "bg-healthy",
    warning: "bg-warning",
    critical: "bg-critical",
  }[status];

  return (
    <span className="relative flex h-2.5 w-2.5">
      <span
        className={`absolute inline-flex h-full w-full animate-ping rounded-full ${colorClass} opacity-75`}
        style={{ animationDuration: "2.5s" }}
      />
      <span className={`relative inline-flex h-2.5 w-2.5 rounded-full ${colorClass}`} />
    </span>
  );
}