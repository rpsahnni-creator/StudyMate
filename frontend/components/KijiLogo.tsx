import Image from "next/image";
import type { CSSProperties } from "react";

type KijiLogoProps = {
  /** Square fallback when width/height omitted (matches mobile LogoMark `size`). */
  size?: number;
  width?: number;
  height?: number;
  className?: string;
  style?: CSSProperties;
  priority?: boolean;
};

/** Kiji logo — same asset & proportions as mobile `LogoMark`. */
export function KijiLogo({
  size,
  width = size ?? 280,
  height = size ?? 100,
  className = "",
  style,
  priority,
}: KijiLogoProps) {
  return (
    <Image
      src="/images/kiji_logo.png"
      alt="Kiji Technology"
      width={width}
      height={height}
      className={`kiji-logo ${className}`.trim()}
      style={style}
      priority={priority}
    />
  );
}
