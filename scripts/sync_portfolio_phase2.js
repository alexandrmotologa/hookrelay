const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const sites = [
  'B:/workgit/mtlg-site',
  'B:/workgit/mtlglabs-space',
];

const hookrelayImages = [
  'images/hookrelay_demo.gif',
  'images/hookrelay_dashboard.png',
  'images/hookrelay_payload_modal.png',
  'images/hookrelay_endpoints.png',
  'images/hookrelay_debugger.png',
  'images/hookrelay_mock_receiver.png',
  'images/hookrelay_customer_portal.png',
  'images/hookrelay_portal_verify.png',
  'images/hookrelay_mobile.png',
];

// Strict integrity assertion: Never include logos in portfolio carousels
for (const img of hookrelayImages) {
  if (img.toLowerCase().includes('logo')) {
    throw new Error('Integrity violation: logo detected in portfolio carousel image array!');
  }
}

for (const site of sites) {
  console.log(`\n=== Updating ${site} ===`);

  // 1. Update project_images_map.json
  const mapPath = path.join(site, 'project_images_map.json');
  if (fs.existsSync(mapPath)) {
    const map = JSON.parse(fs.readFileSync(mapPath, 'utf8'));
    map['hookrelay'] = hookrelayImages;
    fs.writeFileSync(mapPath, JSON.stringify(map, null, 2) + '\n');
    console.log(`✓ Updated ${mapPath}`);
  }

  // 2. Update projects.json
  const projPath = path.join(site, 'projects.json');
  const projects = JSON.parse(fs.readFileSync(projPath, 'utf8'));
  const p = projects.find((x) => x.id === 'hookrelay');
  if (!p) {
    throw new Error(`hookrelay not found in ${projPath}`);
  }

  p.badgeLabs = { text: 'Go Engine', type: 'live' };
  p.badgeSite = { text: 'Webhook Gateway', type: 'live' };
  p.image = 'images/hookrelay_demo.gif';
  p.images = hookrelayImages;
  p.desc = 'Self-hosted outbound webhook delivery engine with Standard Webhooks HMAC-SHA256 signing, exponential backoff retries, and customer delivery portal.';

  fs.writeFileSync(projPath, JSON.stringify(projects, null, 2) + '\n');
  console.log(`✓ Updated hookrelay in ${projPath}`);
}

// 3. Run sync.js and compare_projects.js in B:/workgit/mtlg-site
console.log('\n=== Recompiling and asserting parity in mtlg-site ===');
const syncOutput = execSync('node sync.js', { cwd: 'B:/workgit/mtlg-site', encoding: 'utf8' });
console.log(syncOutput);

const compareOutput = execSync('node compare_projects.js', { cwd: 'B:/workgit/mtlg-site', encoding: 'utf8' });
console.log(compareOutput);

console.log('✓ Phase 2 Portfolio Synchronization complete.');
