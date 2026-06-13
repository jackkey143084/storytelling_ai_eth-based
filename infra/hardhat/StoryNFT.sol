// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract StoryNFT is ERC721, Ownable {
    uint256 public nextId;
    mapping(uint256 => string) public tokenCID;
    mapping(uint256 => bytes32) public checkpointHash;

    event StoryMinted(address indexed owner, uint256 indexed tokenId, string cid, bytes32 checkpoint);

    constructor(string memory name_, string memory symbol_) ERC721(name_, symbol_) {
        nextId = 1;
    }

    function mintStory(address to, string memory cid, bytes32 checkpoint) public onlyOwner returns (uint256) {
        uint256 tokenId = nextId;
        _safeMint(to, tokenId);
        tokenCID[tokenId] = cid;
        checkpointHash[tokenId] = checkpoint;
        emit StoryMinted(to, tokenId, cid, checkpoint);
        nextId += 1;
        return tokenId;
    }

    function tokenURI(uint256 tokenId) public view override returns (string memory) {
        require(_exists(tokenId), "Nonexistent");
        // return IPFS gateway URL — consumer can replace prefix
        return string(abi.encodePacked("ipfs://", tokenCID[tokenId]));
    }
}

