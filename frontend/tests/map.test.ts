import { test, expect } from 'vitest';
import {
  CEREMONY,
  RECEPTION,
  WALKING_ROUTE,
  googleMapsSearchUrl,
  googleMapsDirectionsUrl,
} from '../src/scripts/map';

test('should encode the point as the search query', () => {
  const url = googleMapsSearchUrl({ lat: 55.7048, lng: 13.1965 });

  expect(url).toBe('https://www.google.com/maps/search/?api=1&query=55.7048%2C13.1965');
});

test('should build walking directions from the ceremony to the reception', () => {
  const url = googleMapsDirectionsUrl(CEREMONY, RECEPTION);

  expect(url).toContain(`origin=${CEREMONY.lat}%2C${CEREMONY.lng}`);
  expect(url).toContain(`destination=${RECEPTION.lat}%2C${RECEPTION.lng}`);
  expect(url).toContain('travelmode=walking');
});

// Equirectangular approximation, plenty accurate over a few hundred metres.
function metresBetween(a: { lat: number; lng: number }, b: { lat: number; lng: number }): number {
  const toRad = Math.PI / 180;
  const x = (b.lng - a.lng) * toRad * Math.cos(((a.lat + b.lat) / 2) * toRad);
  const y = (b.lat - a.lat) * toRad;
  return Math.hypot(x, y) * 6_371_000;
}

test('should start the walking route at the ceremony and end it at the reception', () => {
  const first = WALKING_ROUTE[0];
  const last = WALKING_ROUTE[WALKING_ROUTE.length - 1];

  expect(first).toEqual(CEREMONY);
  expect(last).toEqual(RECEPTION);
});

test('should keep consecutive walking route points close together', () => {
  const gaps = WALKING_ROUTE.slice(1).map((point, i) => metresBetween(WALKING_ROUTE[i], point));

  expect(Math.max(...gaps)).toBeLessThan(60);
});
