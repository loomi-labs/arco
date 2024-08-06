export interface Path {
  path: string;
  isAdded: boolean;
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