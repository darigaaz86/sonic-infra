import json
import subprocess
import os

# Path to your Terraform directory
TERRAFORM_DIR = "../terraform"  # adjust as needed

# Run terraform output in the correct dir
output = subprocess.check_output(
    ["terraform", "output", "-json", "node_ips"],
    cwd=TERRAFORM_DIR
)

ips = json.loads(output)
validator_count = 15

with open("inventory.ini", "w") as f:
    f.write("[bootnode]\n")
    f.write(f"rpc-1 ansible_host={ips[validator_count]} is_bootnode=true\n\n")

    f.write("[validators]\n")
    for i in range(validator_count):
        f.write(f"validator-{i+1} ansible_host={ips[i]}\n")
    f.write("\n")

    f.write("[rpcs]\n")
    for i in range(validator_count+1, validator_count+3):
        f.write(f"rpc-{i-validator_count+1} ansible_host={ips[i]}\n")
