// Minimal ambient declarations for the runtime this service uses, so the target
// compiles with `tsc` alone and needs no node_modules. Only what the code below
// actually calls is declared.

declare const process: {
  env: { [key: string]: string | undefined };
  exit(code: number): void;
};

declare const console: {
  log(...args: unknown[]): void;
  error(...args: unknown[]): void;
};

declare function setInterval(fn: () => void, ms: number): number;
declare function clearInterval(id: number): void;
declare function setTimeout(fn: () => void, ms: number): number;

declare module "fs" {
  export function readFileSync(path: string, encoding: string): string;
  export function writeFileSync(path: string, data: string): void;
  export function appendFileSync(path: string, data: string): void;
  export function renameSync(from: string, to: string): void;
  export function chmodSync(path: string, mode: number): void;
}

declare module "path" {
  export function join(...parts: string[]): string;
}

declare module "crypto" {
  export function randomBytes(size: number): { toString(encoding: string): string };
  export function createHash(algorithm: string): {
    update(data: string): { digest(encoding: string): string };
  };
}
