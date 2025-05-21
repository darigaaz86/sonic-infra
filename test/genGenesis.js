const fs = require('fs');
const path = require('path');

// Load all accounts from accounts1.json to accounts5.json
let allAccounts = [];
for (let i = 1; i <= 5; i++) {
  const filename = `accounts${i}.json`;
  const accounts = JSON.parse(fs.readFileSync(filename, 'utf8'));
  allAccounts = allAccounts.concat(accounts);
}

// Load the genesis template
const genesis = JSON.parse(fs.readFileSync('example-genesis.json', 'utf8'));

// Ensure target section exists
if (!Array.isArray(genesis.accounts)) {
  genesis.accounts = [];
}

// Add all accounts with unique global names
let globalIndex = 1;
for (const acc of allAccounts) {
  genesis.accounts.push({
    name: `acc${String(globalIndex).padStart(4, '0')}`, // acc0001, acc0002, ...
    address: acc.address,
    balance: "1000000000000000000000000000" // use string temporarily
  });
  globalIndex++;
}

// Convert to JSON and replace balance string with raw number
let jsonString = JSON.stringify(genesis, null, 2);
jsonString = jsonString.replace(
  /"balance": "1000000000000000000000000000"/g,
  '"balance": 1000000000000000000000000000'
);

// Write the new genesis file
fs.writeFileSync('genesis.json', jsonString, 'utf8');

console.log(`✅ New genesis.json written with ${globalIndex - 1} unique accounts.`);
