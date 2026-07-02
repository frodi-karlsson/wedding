export interface LatLng {
  lat: number;
  lng: number;
}

// Venue coordinates, geocoded via OpenStreetMap Nominatim.
export const CEREMONY: LatLng = { lat: 55.7048, lng: 13.1965 };
export const RECEPTION: LatLng = { lat: 55.7055359, lng: 13.1956671 };

/** Google Maps link that opens a single point as a place/search. */
export function googleMapsSearchUrl({ lat, lng }: LatLng): string {
  return `https://www.google.com/maps/search/?api=1&query=${lat}%2C${lng}`;
}

/** Google Maps walking directions between two points. */
export function googleMapsDirectionsUrl(from: LatLng, to: LatLng): string {
  return (
    `https://www.google.com/maps/dir/?api=1` +
    `&origin=${from.lat}%2C${from.lng}` +
    `&destination=${to.lat}%2C${to.lng}` +
    `&travelmode=walking`
  );
}
