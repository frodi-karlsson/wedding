export interface LatLng {
  lat: number;
  lng: number;
}

// Venue coordinates, geocoded via OpenStreetMap Nominatim.
// The ceremony is on the lawn north of Bosebo kyrka, not in the church itself.
export const CEREMONY: LatLng = { lat: 55.70483, lng: 13.19794 };
export const RECEPTION: LatLng = { lat: 55.7055359, lng: 13.1956671 };

// Walking path between the venues, from OpenStreetMap's foot router
// (routing.openstreetmap.de) via a point south of AF-borgen, which is the way we
// walk. Stored here so the site never calls the router at runtime.
// The venue pins are the endpoints so the drawn line meets the markers.
const WALKING_PATH: [number, number][] = [
  [55.704778, 13.197948],
  [55.704773, 13.197837],
  [55.704762, 13.197747],
  [55.704783, 13.197524],
  [55.704797, 13.197396],
  [55.704831, 13.197309],
  [55.704862, 13.19726],
  [55.704895, 13.197203],
  [55.70496, 13.196921],
  [55.704992, 13.196728],
  [55.705011, 13.196593],
  [55.705009, 13.196545],
  [55.705059, 13.196298],
  [55.704961, 13.196227],
  [55.704933, 13.196207],
  [55.704954, 13.196111],
  [55.704909, 13.196083],
  [55.704867, 13.196058],
  [55.704801, 13.196009],
  [55.704811, 13.19596],
  [55.70484, 13.195814],
  [55.704831, 13.195754],
  [55.704829, 13.195722],
  [55.704827, 13.195691],
  [55.704827, 13.195623],
  [55.704841, 13.195528],
  [55.704875, 13.195363],
  [55.704977, 13.19495],
  [55.704985, 13.194914],
  [55.705031, 13.194949],
  [55.705102, 13.194999],
  [55.70521, 13.195076],
  [55.70529, 13.195132],
  [55.705424, 13.195227],
  [55.705499, 13.19528],
  [55.705607, 13.195356],
];

export const WALKING_ROUTE: LatLng[] = [
  CEREMONY,
  ...WALKING_PATH.map(([lat, lng]) => ({ lat, lng })),
  RECEPTION,
];

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
