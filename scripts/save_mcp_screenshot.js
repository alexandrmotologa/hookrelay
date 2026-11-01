const fs = require('fs');

function saveBase64ToFile(txtPath, outPath) {
  const content = fs.readFileSync(txtPath, 'utf8');
  const base64Index = content.indexOf('base64,');
  if (base64Index === -1) {
    throw new Error('base64 marker not found in ' + txtPath);
  }
  const base64Data = content.slice(base64Index + 7).trim();
  const buffer = Buffer.from(base64Data, 'base64');
  fs.writeFileSync(outPath, buffer);
  console.log(`✓ Saved ${outPath} (${buffer.length} bytes)`);
}

const args = process.argv.slice(2);
if (args.length < 2) {
  console.error('Usage: node save_mcp_screenshot.js <input.txt> <output.png>');
  process.exit(1);
}

saveBase64ToFile(args[0], args[1]);
