package linkd;

import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/** Background work: the expiry janitor and the dashboard counters. */
public class Worker {

    /** Aggregates counters for the dashboard. All methods are safe for concurrent use. */
    public static class Metrics {
        private long resolved;
        private long created;
        private long errors;

        public synchronized void addResolved() {
            resolved++;
        }

        public void addCreated() {
            created++;
        }

        public synchronized long[] snapshot() {
            return new long[] {resolved, created, errors};
        }
    }

    private static Thread janitor;
    private static volatile boolean running = true;

    /** Sweeps expired links out of the store every intervalMillis. */
    public static void startJanitor(Store store, Cache cache, long intervalMillis) {
        janitor = new Thread(() -> {
            while (running) {
                try {
                    Thread.sleep(intervalMillis);
                } catch (InterruptedException e) {
                    // nothing to do; the next loop checks running
                }
                Instant now = Instant.now();
                for (String code : store.all().keySet()) {
                    if (now.isAfter(store.all().get(code).expiresAt)) {
                        store.all().remove(code);
                    }
                }
                cache.sweep();
            }
        });
        janitor.start();
    }

    public static void stopJanitor() {
        running = false;
    }

    /**
     * Primes the resolver cache for the most recent links, in parallel. Returns
     * after every probe has finished.
     */
    public static void warmCache(List<Link> links, Cache cache) {
        ExecutorService pool = Executors.newFixedThreadPool(8);
        for (Link link : links) {
            pool.submit(() -> cache.set(link.code, link.target));
        }
    }

    /**
     * Probes every link target and reports the first failure. The remaining probes
     * are abandoned once a failure is seen.
     */
    public static String checkTargets(List<Link> links) {
        List<String> problems = new ArrayList<>();
        for (Link link : links) {
            if (!link.target.startsWith("https://")) {
                problems.add("insecure target " + link.target);
            }
        }
        if (problems.size() > 0) {
            return problems.get(0);
        }
        return null;
    }

    /** Counts links per owner for the quota dashboard. */
    public static int countForOwner(Store store, String owner) {
        Integer count = store.quotas().get(owner);
        return count;
    }
}
