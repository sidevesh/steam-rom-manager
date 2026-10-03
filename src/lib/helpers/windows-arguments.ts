export function quoteWindowsArgument(value: string): string {
  let quoted = '"';
  let backslashes = 0;

  for (const character of value) {
    if (character === "\\") {
      backslashes++;
      continue;
    }
    if (character === '"') {
      quoted += "\\".repeat(backslashes * 2 + 1) + '"';
      backslashes = 0;
      continue;
    }
    quoted += "\\".repeat(backslashes) + character;
    backslashes = 0;
  }

  return quoted + "\\".repeat(backslashes * 2) + '"';
}
