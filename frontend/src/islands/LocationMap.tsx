import { onCleanup, onMount, untrack, type JSX } from 'solid-js';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import type { Lang } from '../scripts/i18n';
import { translate } from '../scripts/i18n';
import {
  CEREMONY,
  RECEPTION,
  googleMapsSearchUrl,
  googleMapsDirectionsUrl,
  type LatLng,
} from '../scripts/map';

interface LocationMapProps {
  lang: Lang;
}

export function LocationMap(props: LocationMapProps): JSX.Element {
  const lang = untrack(() => props.lang);
  let container: HTMLDivElement | undefined;
  let map: L.Map | undefined;

  onMount(() => {
    if (!container) return;

    map = L.map(container, { scrollWheelZoom: false });
    L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
      maxZoom: 19,
      attribution: '© OpenStreetMap contributors',
    }).addTo(map);

    const icon = L.divIcon({
      className: 'map-pin',
      html: '<span class="map-pin__dot"></span>',
      iconSize: [16, 16],
      iconAnchor: [8, 8],
      popupAnchor: [0, -8],
    });

    const openLabel = translate('location_open_maps', lang);
    const venues: { pos: LatLng; name: string; addr: string }[] = [
      {
        pos: CEREMONY,
        name: translate('location_ceremony_name', lang),
        addr: translate('location_ceremony_addr', lang),
      },
      {
        pos: RECEPTION,
        name: translate('location_reception_name', lang),
        addr: translate('location_reception_addr', lang),
      },
    ];

    for (const v of venues) {
      L.marker([v.pos.lat, v.pos.lng], { icon, title: v.name })
        .addTo(map)
        .bindPopup(
          `<strong>${v.name}</strong><br>${v.addr}<br>` +
            `<a href="${googleMapsSearchUrl(v.pos)}" target="_blank" rel="noopener">${openLabel} ↗</a>`,
        );
    }

    L.polyline(
      [
        [CEREMONY.lat, CEREMONY.lng],
        [RECEPTION.lat, RECEPTION.lng],
      ],
      { color: '#2b436b', weight: 3, opacity: 0.6, dashArray: '4 6' },
    ).addTo(map);

    map.fitBounds(
      [
        [CEREMONY.lat, CEREMONY.lng],
        [RECEPTION.lat, RECEPTION.lng],
      ],
      { padding: [50, 50], maxZoom: 17 },
    );
  });

  onCleanup(() => {
    map?.remove();
    map = undefined;
  });

  return (
    <div class="venue-map-wrap">
      <div
        class="venue-map"
        ref={(el) => {
          container = el;
        }}
        role="img"
        aria-label={translate('location_map_aria', lang)}
      />
      <a
        class="venue-map__cta"
        href={googleMapsDirectionsUrl(CEREMONY, RECEPTION)}
        target="_blank"
        rel="noopener"
      >
        {translate('location_open_maps', lang)} ↗
      </a>
    </div>
  );
}

export default LocationMap;
