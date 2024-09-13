export interface Path {
  path: string;
  isAdded: boolean;
  validationError?: string;
}

/* toPaths takes a string array and returns an array of Path objects */
export function toPaths(isAdded: boolean, paths: string[]): Path[] {
  return paths.map((path) => {
    return {
      path,
      isAdded,
    };
  });
}