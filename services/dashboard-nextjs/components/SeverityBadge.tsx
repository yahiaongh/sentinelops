const SEVERITY_STYLES: Record<string, string> = {
  critical: "bg-critical/10 text-critical border-critical/30",
  warning: "bg-warning/10 text-warning border-warning/30",
};

export function SeverityBadge({ severity }: { severity: string }) {
  const styles = SEVERITY_STYLES[severity] ?? "bg-muted/10 text-muted border-muted/30";
  return (
    <span
      className={`inline-flex items-center rounded border px-1.5 py-0.5 font-mono text-xs uppercase tracking-wider ${styles}`}
    >
      {severity}
    </span>
  );
}