import { test, expect } from 'vitest';
import {
  CEREMONY,
  RECEPTION,
  googleMapsSearchUrl,
  googleMapsDirectionsUrl,
} from '../src/scripts/map';

test('googleMapsSearchUrl encodes the point as query', () => {
  expect(googleMapsSearchUrl({ lat: 55.7048, lng: 13.1965 })).toBe(
    'https://www.google.com/maps/search/?api=1&query=55.7048%2C13.1965',
  );
});

test('googleMapsDirectionsUrl builds walking directions between the venues', () => {
  const url = googleMapsDirectionsUrl(CEREMONY, RECEPTION);
  expect(url).toContain('origin=55.7048%2C13.1965');
  expect(url).toContain('destination=55.7055359%2C13.1956671');
  expect(url).toContain('travelmode=walking');
});
