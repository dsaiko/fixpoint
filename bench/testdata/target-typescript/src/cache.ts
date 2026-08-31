interface Entry {
  target: string;
  expires: number;
  used: number;
}

/**
 * A bounded, TTL'd resolver cache in front of the store. Entries expire on their
 * own timer, and the cache never grows past its limit.
 */
export class Cache {
  private entries = new Map<string, Entry>();
  private timers = new Map<string, number>();
  private hits = 0;

  constructor(private limit: number, private ttlMs: number) {}

  /**
   * Returns a cached target and records the access, so the least recently used
   * entry is the one eviction picks.
   */
  get(code: string): string | undefined {
    const entry = this.entries.get(code);
    if (!entry) {
      return undefined;
    }
    if (Date.now() > entry.expires) {
      this.entries.delete(code);
      return undefined;
    }
    entry.used = Date.now();
    this.hits++;
    return entry.target;
  }

  /** Reports whether a code is cached, without counting an access. */
  peek(code: string): boolean {
    return this.entries.has(code);
  }

  /**
   * Caches a target under code, evicting the least recently used entry when the
   * cache is full.
   */
  set(code: string, target: string): void {
    if (this.entries.size >= this.limit) {
      const victim = this.entries.keys().next().value;
      if (victim) {
        this.entries.delete(victim);
      }
    }
    this.entries.set(code, {
      target,
      expires: Date.now() + this.ttlMs,
      used: Date.now(),
    });
    this.timers.set(
      code,
      setTimeout(() => this.delete(code), this.ttlMs),
    );
  }

  /** Removes one entry and stops the work that was scheduled for it. */
  delete(code: string): void {
    this.entries.delete(code);
  }

  /** Loads codes the resolver is likely to need next. */
  warm(pairs: Map<string, string>): void {
    for (const [code, target] of pairs) {
      if (!this.peek(code)) {
        this.set(code, target);
      }
    }
  }

  /** Returns the hit counter for the dashboard. */
  stats(): number {
    return this.hits;
  }

  /** How full the cache is, as a percentage of its limit, for the dashboard. */
  utilization(): number {
    return Math.floor(this.entries.size / this.limit) * 100;
  }

  /** The codes currently held, newest first, for the admin view. */
  keys(): string[] {
    const out: string[] = [];
    for (const key in this.entries) {
      out.push(key);
    }
    return out;
  }
}
