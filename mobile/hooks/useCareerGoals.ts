import { useCallback, useEffect, useState } from "react";
import AsyncStorage from "@react-native-async-storage/async-storage";
import {
  FeatureGatedError,
  getMyGoal,
  getSkillGaps,
  getTodayPractice,
  listGoals,
  type CareerGoal,
  type MyGoal,
  type SkillGap,
  type TodayPractice,
} from "../lib/careerGoals";

export interface ResourceState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
  /** true when the backend gated the feature (403). UI shows "Coming Soon". */
  gated: boolean;
  reload: () => void;
}

function cacheKey(name: string): string {
  return `studyapp.careergoals.${name}`;
}

// useCareerResource centralizes loading/error/gated handling and offline caching.
// On a network failure it falls back to the last cached value when available.
function useCareerResource<T>(
  name: string,
  fetcher: () => Promise<T>,
  cache = true
): ResourceState<T> {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [gated, setGated] = useState(false);
  const [tick, setTick] = useState(0);

  const reload = useCallback(() => setTick((t) => t + 1), []);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);
      setGated(false);
      try {
        const result = await fetcher();
        if (cancelled) return;
        setData(result);
        if (cache) {
          void AsyncStorage.setItem(cacheKey(name), JSON.stringify(result));
        }
      } catch (err) {
        if (cancelled) return;
        if (err instanceof FeatureGatedError) {
          setGated(true);
          setData(null);
        } else {
          // Fall back to cached data (offline support) before surfacing error.
          let recovered = false;
          if (cache) {
            try {
              const cached = await AsyncStorage.getItem(cacheKey(name));
              if (cached && !cancelled) {
                setData(JSON.parse(cached) as T);
                recovered = true;
              }
            } catch {
              // ignore cache read failures
            }
          }
          if (!recovered) {
            setError(err instanceof Error ? err.message : "Something went wrong");
          }
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [name, tick]);

  return { data, loading, error, gated, reload };
}

/** useGoals fetches the available career goals catalog. */
export function useGoals(): ResourceState<CareerGoal[]> {
  return useCareerResource<CareerGoal[]>("goals", listGoals);
}

/** useMyGoal fetches the user's active goal (null data when none is selected). */
export function useMyGoal(): ResourceState<MyGoal | null> {
  return useCareerResource<MyGoal | null>("myGoal", async () => {
    try {
      return await getMyGoal();
    } catch (err) {
      // "no active goal" (404) is an empty state, not an error — the dashboard
      // shows the goal picker in that case.
      if (err instanceof Error && !(err instanceof FeatureGatedError) && err.message === "no active goal") {
        return null;
      }
      throw err;
    }
  });
}

/** useTodayPractice fetches (or triggers creation of) today's practice set. */
export function useTodayPractice(): ResourceState<TodayPractice> {
  return useCareerResource<TodayPractice>("todayPractice", getTodayPractice);
}

/** useSkillGaps fetches the user's skill gaps, weakest first. */
export function useSkillGaps(): ResourceState<SkillGap[]> {
  return useCareerResource<SkillGap[]>("skillGaps", getSkillGaps);
}
