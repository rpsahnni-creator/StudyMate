export type BoardId = "ncert" | "cbse" | "icse" | "jharkhand_board" | "bihar_board";

export interface BoardOption {
  id: BoardId;
  label: string;
}

export const BOARD_OPTIONS: BoardOption[] = [
  { id: "ncert", label: "NCERT" },
  { id: "cbse", label: "CBSE" },
  { id: "icse", label: "ICSE" },
  { id: "jharkhand_board", label: "Jharkhand Board" },
  { id: "bihar_board", label: "Bihar Board" },
];

export function boardLabel(id: string): string {
  return BOARD_OPTIONS.find((option) => option.id === id)?.label ?? id;
}
