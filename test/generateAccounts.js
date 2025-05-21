const { Wallet } = require('ethers');
const fs = require('fs');

const accounts = [];

for (let i = 0; i < 100000; i++) {
  const wallet = Wallet.createRandom();
  accounts.push({
    index: i + 1,
    address: wallet.address,
    privateKey: wallet.privateKey,
    mnemonic: wallet.mnemonic.phrase
  });
}

fs.writeFileSync('accountsTest.json', JSON.stringify(accounts, null, 2));

console.log('✅ 5000 accounts saved to accounts.json');
