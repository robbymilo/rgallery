import React, { useEffect, useRef, useState } from 'react';
import Loading from '../components/Loading';
import 'leaflet/dist/leaflet.css';
import 'leaflet.markercluster/dist/MarkerCluster.css';
import 'leaflet.markercluster/dist/MarkerCluster.Default.css';

import L from 'leaflet';
import 'leaflet.markercluster';

type MapItem = [number, number, number];

const Map: React.FC = () => {
  const mapRef = useRef<HTMLDivElement>(null);
  const leafletMapRef = useRef<L.Map | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let map: L.Map;
    let markers: L.MarkerClusterGroup;

    fetch('/api/map')
      .then((res) => res.json())
      .then((data) => {
        const mapItems: MapItem[] = data.mapItems || [];
        // Default view
        let lat = 0,
          lng = 0,
          zoom = 3;
        const params = new URLSearchParams(window.location.search);
        if (params.get('lat') && params.get('lng')) {
          lat = parseFloat(params.get('lat')!);
          lng = parseFloat(params.get('lng')!);
        }
        if (params.get('zoom')) {
          zoom = parseInt(params.get('zoom')!);
        }

        const tileServer = data.tileServer;
        const defaultMap = tileServer === '/api/tiles/{z}/{x}/{y}.png';
        const maxZoom = defaultMap ? 4 : 19;

        if (leafletMapRef.current) {
          leafletMapRef.current.remove();
          leafletMapRef.current = null;
        }
        const mapEl = mapRef.current as unknown as { _leaflet_id?: string };
        if (mapEl && mapEl._leaflet_id) {
          delete mapEl._leaflet_id;
        }

        map = L.map(mapRef.current!, {
          center: [lat, lng],
          zoom,
        });

        L.tileLayer(tileServer, {
          maxZoom,
        }).addTo(map);

        // Custom marker icon
        const icon = new L.Icon({
          iconUrl: '/static/marker-icon.png',
          iconRetinaUrl: '/static/marker-icon.png',
          shadowUrl: '/static/marker-shadow.png',
          shadowRetinaUrl: '/static/marker-shadow.png',
          iconAnchor: [12, 41],
          popupAnchor: [0, -41],
        });

        markers = L.markerClusterGroup();

        mapItems.forEach(([lat, lng, id]) => {
          const marker = L.marker([lat, lng], { title: String(id), icon });
          marker.bindPopup(`<a href="/media/${id}"><img src="/api/img/${id}/400" width="400" /></a>`, {
            minWidth: 300,
          });
          markers.addLayer(marker);
        });

        map.addLayer(markers);

        if (lat === 0 && lng === 0 && mapItems.length) {
          map.fitBounds(mapItems.map(([lat, lng]) => [lat, lng]));
        }

        map.on('moveend', () => {
          const url = new URL(window.location.href);
          url.searchParams.set('lat', String(map.getCenter().lat));
          url.searchParams.set('lng', String(map.getCenter().lng));
          url.searchParams.set('zoom', String(map.getZoom()));
          history.replaceState(null, document.title, url.href);
        });

        leafletMapRef.current = map;
        setLoading(false);
      });

    return () => {
      if (leafletMapRef.current) {
        leafletMapRef.current.remove();
        leafletMapRef.current = null;
      }
    };
  }, []);

  return (
    <div className="relative h-[calc(100vh-64px)] w-full">
      {loading && <Loading />}
      <div ref={mapRef} className="h-full w-full" id="map" />
    </div>
  );
};

export default Map;
