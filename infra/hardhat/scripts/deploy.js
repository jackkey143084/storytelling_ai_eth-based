async function main() {
  const [deployer] = await ethers.getSigners();
  console.log("Deploying with:", deployer.address);
  const Story = await ethers.getContractFactory("StoryNFT");
  const story = await Story.deploy("StoryNFT", "STR");
  await story.deployed();
  console.log("StoryNFT deployed to:", story.address);
  // print ABI path consumer can use
  console.log("ABI available in artifacts/contracts/StoryNFT.sol/StoryNFT.json");
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
