async function main() {
  const [owner, user] = await ethers.getSigners();
  const Story = await ethers.getContractFactory("StoryNFT");
  const story = await Story.attach("PUT_DEPLOYED_ADDRESS_HERE");
  const tx = await story.mintStory(user.address, "QmExampleCid", ethers.utils.formatBytes32String("checkpoint"));
  await tx.wait();
  console.log("Mint done");
}

main().catch(console.error);
