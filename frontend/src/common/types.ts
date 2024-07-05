export interface Directory {
  path: string;
  isAdded: boolean;
}

/* pathToDirectory takes a string array and returns an array of Directory objects */
export function pathToDirectory(isAdded: boolean, paths: string[]): Directory[] {
  return paths.map((path) => {
    return {
      path,
      isAdded,
    };
  });
}