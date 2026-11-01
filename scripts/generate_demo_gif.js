const puppeteer = require('puppeteer-core');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const CHROME_PATH = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const FRAMES_DIR = path.join(__dirname, '..', 'frames');
const OUTPUT_GIF = path.join(__dirname, '..', 'docs', 'images', 'hookrelay_demo.gif');

function delay(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

async function captureDemo() {
  if (fs.existsSync(FRAMES_DIR)) {
    fs.rmSync(FRAMES_DIR, { recursive: true, force: true });
  }
  fs.mkdirSync(FRAMES_DIR, { recursive: true });

  console.log('Launching Chrome via puppeteer-core...');
  const browser = await puppeteer.launch({
    executablePath: CHROME_PATH,
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1280,800'],
    defaultViewport: { width: 1280, height: 800 },
  });

  const page = await browser.newPage();
  let frameIdx = 0;

  async function snap(count = 1) {
    for (let c = 0; c < count; c++) {
      const filename = `frame_${String(frameIdx++).padStart(3, '0')}.png`;
      await page.screenshot({ path: path.join(FRAMES_DIR, filename) });
    }
  }

  console.log('1. Navigating to Dashboard...');
  await page.goto('http://localhost:9770', { waitUntil: 'domcontentloaded' });
  await delay(1200);
  await snap(6); // 6 frames = ~1.2s on dashboard

  console.log('2. Opening Test Simulator modal...');
  await page.evaluate(() => {
    const pubBtn = Array.from(document.querySelectorAll('button')).find((b) => b.innerText.includes('Publish'));
    if (pubBtn) pubBtn.click();
  });
  await delay(400);
  await snap(5); // ~1.0s on simulator modal

  console.log('3. Triggering webhook dispatch...');
  await page.evaluate(() => {
    const dispatchBtn = Array.from(document.querySelectorAll('button')).find((b) => b.innerText.includes('Publish Webhook') || b.innerText.includes('Dispatch') || b.innerText.includes('Send Webhook'));
    if (dispatchBtn) dispatchBtn.click();
  });
  await delay(800);
  await snap(4); // 0.8s on confirmation

  console.log('4. Returning to Deliveries and inspecting first row...');
  await page.evaluate(() => {
    // Close simulator if still open
    const closeBtn = document.querySelector('button[title="Close"], .fixed button svg.lucide-x')?.closest('button');
    if (closeBtn) closeBtn.click();
  });
  await delay(500);
  await snap(3);

  // Click on first delivery row to inspect payload
  await page.evaluate(() => {
    const firstRowBtn = document.querySelector('tbody tr:first-child button');
    if (firstRowBtn) firstRowBtn.click();
  });
  await delay(500);
  await snap(7); // ~1.4s on payload drawer

  console.log('5. Switching to Response tab in drawer...');
  await page.evaluate(() => {
    const resTab = Array.from(document.querySelectorAll('button')).find((b) => b.innerText.includes('Returned Response'));
    if (resTab) resTab.click();
  });
  await delay(400);
  await snap(5);

  console.log('6. Closing drawer and switching to Endpoints...');
  await page.evaluate(() => {
    const closeBtn = Array.from(document.querySelectorAll('button')).find((b) => b.innerText === '✕' || b.innerHTML.includes('lucide-x'));
    if (closeBtn) closeBtn.click();
    const epTab = Array.from(document.querySelectorAll('button')).find((b) => b.innerText.includes('Endpoints') || b.getAttribute('title') === 'Endpoints');
    if (epTab) epTab.click();
  });
  await delay(600);
  await snap(6); // ~1.2s on endpoints

  console.log('7. Switching to Signature Debugger...');
  await page.evaluate(() => {
    const dbgTab = Array.from(document.querySelectorAll('button')).find((b) => b.innerText.includes('Signature Debugger') || b.getAttribute('title') === 'Signature Debugger');
    if (dbgTab) dbgTab.click();
  });
  await delay(500);
  // Click verify
  await page.evaluate(() => {
    const verifyBtn = Array.from(document.querySelectorAll('button')).find((b) => b.innerText.includes('Verify & Debug'));
    if (verifyBtn) verifyBtn.click();
  });
  await delay(500);
  await snap(6); // ~1.2s on debugger

  console.log('8. Switching to Mock Receiver...');
  await page.evaluate(() => {
    const mockTab = Array.from(document.querySelectorAll('button')).find((b) => b.innerText.includes('Mock Receiver') || b.getAttribute('title') === 'Mock Receiver');
    if (mockTab) mockTab.click();
  });
  await delay(600);
  await snap(6); // ~1.2s on mock receiver

  await browser.close();
  console.log(`Captured ${frameIdx} total frames in ${FRAMES_DIR}`);

  console.log('Encoding animated demo GIF via ffmpeg...');
  const palettePath = path.join(FRAMES_DIR, 'palette.png');
  // 1. Generate palette for crisp dark-mode color quantization
  execSync(`ffmpeg -y -framerate 5 -i "${FRAMES_DIR}\\frame_%03d.png" -vf "fps=5,scale=1000:-1:flags=lanczos,palettegen" "${palettePath}"`);
  // 2. Use palette to generate the GIF
  execSync(`ffmpeg -y -framerate 5 -i "${FRAMES_DIR}\\frame_%03d.png" -i "${palettePath}" -lavfi "fps=5,scale=1000:-1:flags=lanczos [x]; [x][1:v] paletteuse=dither=bayer:bayer_scale=3" "${OUTPUT_GIF}"`);

  const stat = fs.statSync(OUTPUT_GIF);
  console.log(`✓ Successfully created ${OUTPUT_GIF} (${(stat.size / 1024 / 1024).toFixed(2)} MB)`);
}

captureDemo().catch((err) => {
  console.error('Demo capture failed:', err);
  process.exit(1);
});
