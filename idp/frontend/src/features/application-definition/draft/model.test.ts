import { afterEach, vi } from 'vitest';
import { newId } from './model';

const UUID_V4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const randomUUID = crypto.randomUUID;

afterEach(() => {
  crypto.randomUUID = randomUUID;
});

test('uses crypto.randomUUID when available', () => {
  const spy = vi.spyOn(crypto, 'randomUUID');
  expect(newId()).toMatch(UUID_V4);
  expect(spy).toHaveBeenCalled();
});

test('still returns unique UUIDs without crypto.randomUUID (page served over plain HTTP)', () => {
  // @ts-expect-error simulate a context where the function is missing
  crypto.randomUUID = undefined;
  const ids = new Set(Array.from({ length: 200 }, newId));
  expect(ids.size).toBe(200);
  for (const id of ids) expect(id).toMatch(UUID_V4);
});
