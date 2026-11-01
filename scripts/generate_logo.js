const fs = require('fs');
const path = require('path');
const { Resvg } = require('@resvg/resvg-js');

function buildLogoSvg() {
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="1024" height="1024">
  <defs>
    <!-- Clip to squircle boundary -->
    <clipPath id="squircle-clip">
      <rect x="24" y="24" width="976" height="976" rx="220" />
    </clipPath>

    <!-- Linear Gradients -->
    <linearGradient id="cyan-glow" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#00f5ff"/>
      <stop offset="100%" stop-color="#0284c7"/>
    </linearGradient>

    <linearGradient id="relay-emerald" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#34d399"/>
      <stop offset="100%" stop-color="#059669"/>
    </linearGradient>

    <linearGradient id="amber-cere" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#fbbf24"/>
      <stop offset="100%" stop-color="#d97706"/>
    </linearGradient>

    <linearGradient id="core-glow" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#38bdf8"/>
      <stop offset="50%" stop-color="#00f5ff"/>
      <stop offset="100%" stop-color="#10b981"/>
    </linearGradient>

    <!-- Container Drop Shadow -->
    <filter id="subtle-shadow" x="-15%" y="-15%" width="130%" height="130%">
      <feDropShadow dx="0" dy="16" stdDeviation="20" flood-color="#0f172a" flood-opacity="0.16" />
    </filter>

    <filter id="core-bloom" x="-30%" y="-30%" width="160%" height="160%">
      <feGaussianBlur stdDeviation="6" result="blur" />
      <feComposite in="SourceGraphic" in2="blur" operator="over" />
    </filter>
  </defs>

  <!-- Luxury White Squircle Container -->
  <rect x="24" y="24" width="976" height="976" rx="220" fill="#ffffff" stroke="#e2e8f0" stroke-width="6" />

  <g clip-path="url(#squircle-clip)">
    <g transform="translate(512, 512)" filter="url(#subtle-shadow)">

      <!-- Hexagonal Architectural Gateway Frame -->
      <polygon points="
        0,-400
        346,-200
        346,200
        0,400
        -346,200
        -346,-200
      " fill="none" stroke="#0f172a" stroke-width="36" stroke-linejoin="round" />

      <!-- Inner Dashed Telemetry Ring -->
      <polygon points="
        0,-364
        315,-182
        315,182
        0,364
        -315,182
        -315,-182
      " fill="none" stroke="#00f5ff" stroke-width="4" opacity="0.45" stroke-dasharray="16, 12" />

      <!-- Corner Hardware Anchor Nodes -->
      <circle cx="0" cy="-400" r="8" fill="#00f5ff" />
      <circle cx="346" cy="-200" r="8" fill="#00f5ff" />
      <circle cx="346" cy="200" r="8" fill="#00f5ff" />
      <circle cx="0" cy="400" r="8" fill="#10b981" />
      <circle cx="-346" cy="200" r="8" fill="#00f5ff" />
      <circle cx="-346" cy="-200" r="8" fill="#00f5ff" />

      <!-- ======================================================== -->
      <!-- 100% WATERTIGHT SOLID SILHOUETTE BASE (No Background Leaks) -->
      <!-- ======================================================== -->
      <path d="
        M 0,-300
        L -45,-265
        L -95,-210
        L -85,-140
        L -135,-80
        L -225,-20
        L -290,70
        L -265,190
        L -190,275
        L -115,335
        L -45,355
        L 0,365
        L 45,355
        L 115,335
        L 190,275
        L 265,190
        L 290,70
        L 225,-20
        L 135,-80
        L 85,-140
        L 95,-210
        L 45,-265
        Z
      " fill="#060911" />

      <!-- ======================================================== -->
      <!-- FALCON CROWN & STREAMLINED RAPTOR HEAD                    -->
      <!-- ======================================================== -->

      <!-- Aerodynamic Crown Spire (Pointed Center Apex) -->
      <polygon points="0,-300 -45,-265 -25,-215 0,-240" fill="#1e293b" />
      <polygon points="0,-300 45,-265 25,-215 0,-240" fill="#334155" />

      <!-- Swept Crest Planes (Nape Feathers) -->
      <polygon points="-45,-265 -95,-210 -55,-175 -25,-215" fill="#0f172a" />
      <polygon points="45,-265 95,-210 55,-175 25,-215" fill="#475569" />

      <!-- Occipital Nape Plates -->
      <polygon points="-95,-210 -85,-140 -45,-145 -55,-175" fill="#1e293b" />
      <polygon points="95,-210 85,-140 45,-145 55,-175" fill="#334155" />

      <!-- Forehead Central Median Plane -->
      <polygon points="0,-240 -25,-215 -35,-165 0,-185" fill="#334155" />
      <polygon points="0,-240 25,-215 35,-165 0,-185" fill="#475569" />

      <!-- Falcon Eyebrow Ridge (Aggressive Supraorbital Shelf) -->
      <polygon points="0,-185 -35,-165 -75,-140 -35,-125 0,-140" fill="#1e293b" />
      <polygon points="0,-185 35,-165 75,-140 35,-125 0,-140" fill="#334155" />

      <!-- Eye Socket Cavity (Deep Shadow) -->
      <polygon points="-75,-140 -35,-125 -30,-90 -70,-100" fill="#050811" />
      <polygon points="75,-140 35,-125 30,-90 70,-100" fill="#050811" />

      <!-- Intense Predator Almond Eyes (Slanted Upward, Electric Cyan) -->
      <polygon points="-72,-132 -44,-118 -36,-98 -62,-112" fill="url(#cyan-glow)" />
      <polygon points="-56,-124 -46,-116 -42,-106 -51,-114" fill="#ffffff" opacity="0.95" />

      <polygon points="72,-132 44,-118 36,-98 62,-112" fill="url(#cyan-glow)" />
      <polygon points="56,-124 46,-116 42,-106 51,-114" fill="#ffffff" opacity="0.95" />

      <!-- Cheek / Malar Plumage Plates -->
      <polygon points="-85,-140 -135,-80 -85,-65 -45,-145" fill="#0f172a" />
      <polygon points="85,-140 135,-80 85,-65 45,-145" fill="#1e293b" />

      <polygon points="-70,-100 -30,-90 -45,-40 -85,-65" fill="#1e293b" />
      <polygon points="70,-100 30,-90 45,-40 85,-65" fill="#334155" />

      <!-- ======================================================== -->
      <!-- HOOKED RAPTOR BEAK & GOLDEN CERE                         -->
      <!-- ======================================================== -->

      <!-- Golden Amber Cere (Nostril Base) -->
      <polygon points="0,-140 -20,-115 0,-105" fill="url(#amber-cere)" />
      <polygon points="0,-140 20,-115 0,-105" fill="url(#amber-cere)" />

      <!-- Upper Beak Bridge (Culmen) -->
      <polygon points="0,-105 -20,-115 -16,-45 0,-10" fill="#475569" />
      <polygon points="0,-105 20,-115 16,-45 0,-10" fill="#64748b" />

      <!-- Sharp Down-Curving Beak Talon -->
      <polygon points="0,-10 -16,-45 -10,-5 0,26" fill="#0f172a" />
      <polygon points="0,-10 16,-45 10,-5 0,26" fill="#1e293b" />

      <!-- Lower Mandible & Recessed Chin -->
      <polygon points="0,26 -10,-5 -5,40 0,44" fill="#090d16" />
      <polygon points="0,26 10,-5 5,40 0,44" fill="#0f172a" />

      <!-- Chin & Neck Chevron Plates -->
      <polygon points="0,44 -45,-40 -25,70 0,82" fill="#1e293b" />
      <polygon points="0,44 45,-40 25,70 0,82" fill="#334155" />

      <polygon points="-45,-40 -85,-65 -65,30 -25,70" fill="#0f172a" />
      <polygon points="45,-40 85,-65 65,30 25,70" fill="#1e293b" />

      <!-- ======================================================== -->
      <!-- AERODYNAMIC FOLDED WINGS & SCAPULARS                     -->
      <!-- ======================================================== -->

      <!-- Shoulder Caps -->
      <polygon points="-135,-80 -225,-20 -165,20 -85,-65" fill="#1e293b" />
      <polygon points="135,-80 225,-20 165,20 85,-65" fill="#334155" />

      <polygon points="-225,-20 -290,70 -220,80 -165,20" fill="#0f172a" />
      <polygon points="225,-20 290,70 220,80 165,20" fill="#1e293b" />

      <!-- Primary Flight Feathers - Outer Edge -->
      <polygon points="-290,70 -265,190 -200,165 -220,80" fill="#1e293b" />
      <polygon points="-265,190 -190,275 -145,225 -200,165" fill="#0f172a" />
      <polygon points="-190,275 -115,335 -85,280 -145,225" fill="#1e293b" />

      <polygon points="290,70 265,190 200,165 220,80" fill="#334155" />
      <polygon points="265,190 190,275 145,225 200,165" fill="#1e293b" />
      <polygon points="190,275 115,335 85,280 145,225" fill="#334155" />

      <!-- Secondary Wing Coverts -->
      <polygon points="-165,20 -220,80 -150,115 -105,40" fill="#1e293b" />
      <polygon points="-220,80 -200,165 -140,170 -150,115" fill="#334155" />
      <polygon points="-200,165 -145,225 -95,205 -140,170" fill="#0f172a" />

      <polygon points="165,20 220,80 150,115 105,40" fill="#334155" />
      <polygon points="220,80 200,165 140,170 150,115" fill="#475569" />
      <polygon points="200,165 145,225 95,205 140,170" fill="#1e293b" />

      <!-- Tail Fan / Rudder Center -->
      <polygon points="-115,335 -45,355 0,365 0,305 -55,285" fill="#090d16" />
      <polygon points="115,335 45,355 0,365 0,305 55,285" fill="#0f172a" />
      <polygon points="-55,285 0,305 55,285 0,255" fill="#1e293b" />

      <!-- ======================================================== -->
      <!-- BREAST ARMOR & DISPATCH RELAY CORE                       -->
      <!-- ======================================================== -->

      <!-- Upper Chest Plate -->
      <polygon points="0,82 -25,70 -60,105 0,118" fill="#1e293b" />
      <polygon points="0,82 25,70 60,105 0,118" fill="#334155" />

      <polygon points="-25,70 -105,40 -90,105 -60,105" fill="#0f172a" />
      <polygon points="25,70 105,40 90,105 60,105" fill="#1e293b" />

      <!-- Mid Chest Ribs -->
      <polygon points="-60,105 -90,105 -75,165 -35,150" fill="#1e293b" />
      <polygon points="60,105 90,105 75,165 35,150" fill="#334155" />

      <polygon points="-35,150 -75,165 -50,220 0,205" fill="#0f172a" />
      <polygon points="35,150 75,165 50,220 0,205" fill="#1e293b" />

      <!-- Belly Keel -->
      <polygon points="0,205 -50,220 0,255" fill="#090d16" />
      <polygon points="0,205 50,220 0,255" fill="#0f172a" />

      <!-- ======================================================== -->
      <!-- THE OUTBOUND RELAY NEXUS (The Dispatch Waypoint Core)    -->
      <!-- ======================================================== -->

      <!-- Outer Diamond Housing -->
      <polygon points="
        0,110
        42,140
        42,190
        0,220
        -42,190
        -42,140
      " fill="#060911" stroke="#00f5ff" stroke-width="2.5" />

      <!-- Radiant Telemetry Crystal -->
      <polygon points="
        0,118
        34,144
        34,186
        0,212
        -34,186
        -34,144
      " fill="url(#core-glow)" filter="url(#core-bloom)" />

      <!-- Dispatch Relay Vector Arrow -->
      <polygon points="0,128 15,150 5,150 5,176 -5,176 -5,150 -15,150" fill="#ffffff" />
      <circle cx="0" cy="194" r="4.5" fill="#ffffff" />

      <!-- Symmetrical Cyan Telemetry Traces -->
      <line x1="-42" y1="165" x2="-95" y2="165" stroke="#00f5ff" stroke-width="2" opacity="0.5" stroke-dasharray="4, 4" />
      <line x1="42" y1="165" x2="95" y2="165" stroke="#00f5ff" stroke-width="2" opacity="0.5" stroke-dasharray="4, 4" />
      <circle cx="-95" cy="165" r="3" fill="#00f5ff" />
      <circle cx="95" cy="165" r="3" fill="#00f5ff" />

      <!-- Vertical Spinal Telemetry Seams -->
      <line x1="0" y1="-300" x2="0" y2="-185" stroke="#00f5ff" stroke-width="2" opacity="0.4" />
      <line x1="0" y1="220" x2="0" y2="255" stroke="#10b981" stroke-width="2.5" opacity="0.7" />

    </g>
  </g>
</svg>`;
}

async function renderLogo() {
  const outputDir = path.join(__dirname, '..', 'docs', 'images');
  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true });
  }

  const svg = buildLogoSvg();
  const svgPath = path.join(outputDir, 'hookrelay_logo.svg');
  const pngPath = path.join(outputDir, 'hookrelay_logo.png');

  fs.writeFileSync(svgPath, svg);
  const resvg = new Resvg(svg, {
    fitTo: {
      mode: 'width',
      value: 1024,
    },
  });
  const pngData = resvg.render();
  const pngBuffer = pngData.asPng();
  fs.writeFileSync(pngPath, pngBuffer);
  console.log(`✓ Successfully rendered refined hookrelay_logo.svg (${svg.length} bytes) and hookrelay_logo.png (${pngBuffer.length} bytes) at 1024x1024`);
}

renderLogo().catch(err => {
  console.error('Error rendering logo:', err);
  process.exit(1);
});
