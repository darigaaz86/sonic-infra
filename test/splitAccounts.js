const fs = require('fs');

// Load the original accounts file
const allAccounts = JSON.parse(fs.readFileSync('accountsTest.json', 'utf8'));

// Determine split size
const numFiles = 10;
const chunkSize = Math.ceil(allAccounts.length / numFiles);

// Split and write to individual files
for (let i = 0; i < numFiles; i++) {
  const start = i * chunkSize;
  const end = start + chunkSize;
  const chunk = allAccounts.slice(start, end);

  const filename = `accounts${i + 1}.json`;
  fs.writeFileSync(filename, JSON.stringify(chunk, null, 2), 'utf8');
  console.log(`✅ Wrote ${chunk.length} accounts to ${filename}`);
}
