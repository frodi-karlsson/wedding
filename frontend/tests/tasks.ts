import { Task } from 'anabranch';
import type { HttpError } from '@anabranch/web-client';

// Every promise a mocked task hands to a component, so a test can wait for
// exactly the API calls the component made instead of polling the DOM.
const inFlight: Promise<unknown>[] = [];

function track<T>(promise: Promise<T>): Promise<T> {
  inFlight.push(promise);
  return promise;
}

export function mockTask<T>(value: T): Task<T, HttpError> {
  return Task.of(() => track(Promise.resolve(value)));
}

export function mockFailingTask<T>(error: Error): Task<T, HttpError> {
  return Task.of(() => track(Promise.reject(error)));
}

// Mocked tasks never touch timers, so one macrotask turn runs every microtask
// chained on a settled task, including the component's own then/catch/await.
function drainMicrotasks(): Promise<void> {
  return new Promise((resolve) => {
    globalThis.setTimeout(resolve, 0);
  });
}

/**
 * Resolve once every task the component has run has settled and its handlers
 * have finished. Tasks started by those handlers (a refresh after a save, say)
 * are waited for too.
 */
export async function settleTasks(): Promise<void> {
  while (inFlight.length > 0) {
    const batch = inFlight.splice(0);
    await Promise.allSettled(batch);
    await drainMicrotasks();
  }
}
