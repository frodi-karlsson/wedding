import { test, expect } from 'vitest';
import {
  CEREMONY,
  RECEPTION,
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
