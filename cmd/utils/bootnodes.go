// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package utils

import "github.com/ethereumproject/go-ethereum/p2p/discover"

// HomesteadBootNodes are the enode URLs of the P2P bootstrap nodes running on
// the Homestead network.
var HomesteadBootNodes = []*discover.Node{
	// ETC Volunteer Go Bootnodes
	// EPROJECT Upgrade by supplying a default list, parsing oracle & random selection
	discover.MustParseNode("enode://08c7ee6a4f861ff0664a49532bcc86de1363acd608999d1b76609bb9bc278649906f069057630fd9493924a368b5d1dc9b8f8bf13ac26df72512f6d1fabd8c95@45.32.7.81:30303"),      //dan-
	discover.MustParseNode("enode://e809c4a2fec7daed400e5e28564e23693b23b2cc5a019b612505631bbe7b9ccf709c1796d2a3d29ef2b045f210caf51e3c4f5b6d3587d43ad5d6397526fa6179@174.112.32.157:30303"),  //bar-
	discover.MustParseNode("enode://687be94c3a7beaa3d2fde82fa5046cdeb3e8198354e05b29d6e0d4e276713e3707ac10f784a7904938b06b46c764875c241b0337dd853385a4d8bfcbf8190647@95.183.51.229:30303"),   //tre-
	discover.MustParseNode("enode://6e538e7c1280f0a31ff08b382db5302480f775480b8e68f8febca0ceff81e4b19153c6f8bf60313b93bef2cc34d34e1df41317de0ce613a201d1660a788a03e2@52.206.67.235:30303"),   //ela-
	discover.MustParseNode("enode://ca5ae4eca09ba6787e29cf6d86f7634d07aae6b9e6317a59aff675851c0bf445068173208cf8ef7f5cd783d8e29b85b2fa3fa358124cf0546823149724f9bde1@138.68.1.16:30303"),     //ige-
	discover.MustParseNode("enode://217ebe27e89bf4fec8ce06509323ff095b1014378deb75ab2e5f6759a4e8750a3bd8254b8c6833136e4d5e58230d65ee8ab34a5db5abf0640408c4288af3c8a7@188.138.1.237:30303"),   //lec-
	discover.MustParseNode("enode://fa20444ef991596ce99b81652ac4e61de1eddc4ff21d3cd42762abd7ed47e7cf044d3c9ccddaf6035d39725e4eb372806787829ccb9a08ec7cb71883cb8c3abd@50.149.116.182:30303"),  //Vla-
	discover.MustParseNode("enode://4bd6a4df3612c718333eb5ea7f817923a8cdf1bed89cee70d1710b45a0b6b77b2819846440555e451a9b602ad2efa2d2facd4620650249d8468008946887820a@71.178.232.20:30304"),   //NoM-
	discover.MustParseNode("enode://921cf8e4c345fe8db913c53964f9cadc667644e7f20195a0b7d877bd689a5934e146ff2c2259f1bae6817b6585153a007ceb67d260b720fa3e6fc4350df25c7f@51.255.49.170:30303"),   //jai-
	discover.MustParseNode("enode://ffea3b01c000cdd89e1e9229fea3e80e95b646f9b2aa55071fc865e2f19543c9b06045cc2e69453e6b78100a119e66be1b5ad50b36f2ffd27293caa28efdd1b2@128.199.93.177:3030"),   //pys-
	discover.MustParseNode("enode://ee3da491ce6a155eb132708eb0e8d04b0637926ec0ae1b79e63fc97cb9fc3818f49250a0ae0d7f79ed62b66ec677f408c4e01741504dc7a051e274f1e803d454@91.121.65.179:40404"),   //mso-
	discover.MustParseNode("enode://48e063a6cf5f335b1ef2ed98126bf522cf254396f850c7d442fe2edbbc23398787e14cd4de7968a00175a82762de9cbe9e1407d8ccbcaeca5004d65f8398d759@159.203.255.59:30303"),  //mik-
	discover.MustParseNode("enode://42d8f29d1db5f4b2947cd5c3d76c6d0d3697e6b9b3430c3d41e46b4bb77655433aeedc25d4b4ea9d8214b6a43008ba67199374a9b53633301bca0cd20c6928ab@104.155.176.151:30303"), //boot.gastracker.io
	discover.MustParseNode("enode://814920f1ec9510aa9ea1c8f79d8b6e6a462045f09caa2ae4055b0f34f7416fca6facd3dd45f1cf1673c0209e0503f02776b8ff94020e98b6679a0dc561b4eba0@104.154.136.117:30303"), //boot1.etcdevteam.com
	discover.MustParseNode("enode://72e445f4e89c0f476d404bc40478b0df83a5b500d2d2e850e08eb1af0cd464ab86db6160d0fde64bd77d5f0d33507ae19035671b3c74fec126d6e28787669740@104.198.71.200:30303"),  //boot2.etcdevteam.com

	// Pending & Not Resolving
	//discover.MustParseNode("enode://b61123cc535d6bac44f9e6ff8637a30a10198f80b5582148dcd84ef8039a4b90e326bb7f6964588a46bcf1ccd8e8bba65db514fc72e3026ff13b20959f45f654@etc.naphex.rocks:30303"), // nap-
	//discover.MustParseNode("enode://d55f15f28317c21c359c8f62b93b7059aa2fcd586c0b0d431f97c4b8f27ee8f58fbe060b72eff95790b7ecd34c2a9b02458a783e61d8ec2aa37cdad6b0fc6d9a@node1.ethc.io:30303"),  //phr-
}

// TestNetBootNodes are the enode URLs of the P2P bootstrap nodes running on the
// Morden test network.
var TestNetBootNodes = []*discover.Node{
	// ETC Nodes
	discover.MustParseNode("enode://fb28713820e718066a2f5df6250ae9d07cff22f672dbf26be6c75d088f821a9ad230138ba492c533a80407d054b1436ef18e951bb65e6901553516c8dffe8ff0@104.155.176.151:30304"), //boot.gastracker.io
	discover.MustParseNode("enode://afdc6076b9bf3e7d3d01442d6841071e84c76c73a7016cb4f35c0437df219db38565766234448f1592a07ba5295a867f0ce87b359bf50311ed0b830a2361392d@104.154.136.117:30403"), //boot1.etcdevteam.com
	discover.MustParseNode("enode://21101a9597b79e933e17bc94ef3506fe99a137808907aa8fefa67eea4b789792ad11fb391f38b00087f8800a2d3dff011572b62a31232133dd1591ac2d1502c8@104.198.71.200:30403"),  //boot2.etcdevteam.com

	// ETH/DEV Go Bootnodes
	discover.MustParseNode("enode://e4533109cc9bd7604e4ff6c095f7a1d807e15b38e9bfeb05d3b7c423ba86af0a9e89abbf40bd9dde4250fef114cd09270fa4e224cbeef8b7bf05a51e8260d6b8@94.242.229.4:40404"),
	discover.MustParseNode("enode://8c336ee6f03e99613ad21274f269479bf4413fb294d697ef15ab897598afb931f56beb8e97af530aee20ce2bcba5776f4a312bc168545de4d43736992c814592@94.242.229.203:30303"),

	// ETH/DEV Cpp Bootnodes
}
