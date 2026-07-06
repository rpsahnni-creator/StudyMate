import type { CSSProperties } from "react";

const sizes: Record<string, CSSProperties> = {
  sm: { fontSize: 17, letterSpacing: "0" },
  md: { fontSize: 22, letterSpacing: "0" },
  lg: { fontSize: "clamp(34px, 6vw, 52px)", letterSpacing: "0" },
};

export function StudyMateWord({
  size = "md",
  style,
}: {
  size?: "sm" | "md" | "lg";
  style?: CSSProperties;
}) {
  return (
    <span className="studymate-word" style={{ ...sizes[size], ...style }}>
      <span className="studymate-study">Study</span>
      <span className="studymate-mate">Mate</span>
    </span>
  );
}
