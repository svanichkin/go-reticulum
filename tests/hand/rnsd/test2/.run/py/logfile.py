Exception in thread Thread-4 (reconnect):
Traceback (most recent call last):
  File "/Library/Frameworks/Python.framework/Versions/3.10/lib/python3.10/threading.py", line 1016, in _bootstrap_inner
    self.run()
  File "/Library/Frameworks/Python.framework/Versions/3.10/lib/python3.10/threading.py", line 953, in run
    self._target(*self._args, **self._kwargs)
  File "/Users/alien/Vault/Projects/Self/Golang/Reticulum/go-reticulum/python/RNS/Interfaces/TCPInterface.py", line 298, in reconnect
    RNS.Transport.synthesize_tunnel(self)
  File "/Users/alien/Vault/Projects/Self/Golang/Reticulum/go-reticulum/python/RNS/Transport.py", line 2110, in synthesize_tunnel
    public_key     = RNS.Transport.identity.get_public_key()
AttributeError: 'NoneType' object has no attribute 'get_public_key'
Exception in thread Thread-7 (reconnect):
Traceback (most recent call last):
  File "/Library/Frameworks/Python.framework/Versions/3.10/lib/python3.10/threading.py", line 1016, in _bootstrap_inner
    self.run()
  File "/Library/Frameworks/Python.framework/Versions/3.10/lib/python3.10/threading.py", line 953, in run
    self._target(*self._args, **self._kwargs)
  File "/Users/alien/Vault/Projects/Self/Golang/Reticulum/go-reticulum/python/RNS/Interfaces/TCPInterface.py", line 298, in reconnect
    RNS.Transport.synthesize_tunnel(self)
  File "/Users/alien/Vault/Projects/Self/Golang/Reticulum/go-reticulum/python/RNS/Transport.py", line 2110, in synthesize_tunnel
    public_key     = RNS.Transport.identity.get_public_key()
AttributeError: 'NoneType' object has no attribute 'get_public_key'
[2026-04-13 18:10:03] [Debug]    Reticulum running in interpreted mode
[2026-04-13 18:10:03] [Debug]    Started shared instance interface: Shared Instance[37434]
[2026-04-13 18:10:03] [Debug]    Cleaning ratchets...
[2026-04-13 18:10:03] [Extra]    Cleaning resource and packet caches...
[2026-04-13 18:10:03] [Verbose]  Bringing up system interfaces...
[2026-04-13 18:10:03] [Debug]    Establishing TCP connection for TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]...
[2026-04-13 18:10:03] [Debug]    TCP connection for TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] established
[2026-04-13 18:10:03] [Debug]    TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] hardware MTU set to 8192
[2026-04-13 18:10:03] [Debug]    Establishing TCP connection for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]...
[2026-04-13 18:10:03] [Error]    Initial connection for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] could not be established: [Errno 8] nodename nor servname provided, or not known
[2026-04-13 18:10:03] [Error]    Leaving unconnected and retrying connection in 5 seconds.
[2026-04-13 18:10:03] [Debug]    TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] hardware MTU set to 8192
[2026-04-13 18:10:03] [Debug]    Establishing TCP connection for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]...
[2026-04-13 18:10:03] [Debug]    TCP connection for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] established
[2026-04-13 18:10:03] [Debug]    TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] hardware MTU set to 8192
[2026-04-13 18:10:03] [Debug]    Establishing TCP connection for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]...
[2026-04-13 18:10:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:10:08] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:08] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:10:08] [Error]    Initial connection for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] could not be established: timed out
[2026-04-13 18:10:08] [Error]    Leaving unconnected and retrying connection in 5 seconds.
[2026-04-13 18:10:08] [Debug]    TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] hardware MTU set to 8192
[2026-04-13 18:10:08] [Debug]    Establishing TCP connection for TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]...
[2026-04-13 18:10:08] [Debug]    TCP connection for TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] established
[2026-04-13 18:10:08] [Debug]    TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] hardware MTU set to 8192
[2026-04-13 18:10:08] [Debug]    Establishing TCP connection for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]...
[2026-04-13 18:10:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:10:13] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:13] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:10:13] [Warning]  An interface error occurred for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965], the contained exception was: 'NoneType' object has no attribute 'get_public_key'
[2026-04-13 18:10:13] [Warning]  Attempting to reconnect...
[2026-04-13 18:10:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 18:10:13] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:13] [Error]    Initial connection for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] could not be established: timed out
[2026-04-13 18:10:13] [Error]    Leaving unconnected and retrying connection in 5 seconds.
[2026-04-13 18:10:13] [Debug]    TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] hardware MTU set to 8192
[2026-04-13 18:10:13] [Verbose]  System interfaces are ready
[2026-04-13 18:10:13] [Debug]    Utilising cryptography backend "openssl, PyCA 46.0.3"
[2026-04-13 18:10:13] [Verbose]  Configuration loaded from /Users/alien/Vault/Projects/Self/Golang/Reticulum/go-reticulum/tests/hand/rnsd/test2/.run/py/config
[2026-04-13 18:10:13] [Verbose]  Loaded 346 known destination from storage
[2026-04-13 18:10:13] [Verbose]  Loaded Transport Identity from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <7aeedc8af83553263107eeac841ccebd> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <2ec8076262ba5be05b6163aec2b54fc5> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <7e193144b02086570fa0b85c6515a57f> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <217319a843e7dc4313b02a3abdf2b6cf> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <6d1319b8154d542d84fee89c9a131c98> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <74bc0ab32aa78578d62fb005f4e6ebdb> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <fe7747a51dc81e4cb341f20de8b18cdc> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <3cf1d67e30b17a7fadff9aaaa6034038> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <29b2ebe588859e48aabf13e97cfe245b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <91cc46a09f3f6131d175dddc4fe97c6a> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <709a6352bcd7e3bd04d8c90e1fd1e647> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f89ede8428bb261e3e2a935dfe920f40> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <979a5adf9bc9721c9146b68dea00e144> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f240978b5ae0ea3b2bd37480f36245a3> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <103eb3c7f35278ba33e7d014e341b3ec> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <13da3e301bfb15dc0d0499859e0e7cfe> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <d8fcdfad947acf53a0f0f8ea7cc2dd07> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <02aaf088472435718061211d3752c8ed> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <e345f6220682e127cab52c3387436778> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f202ae6541f5e69c204d0b2bcbfcd273> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <60dcab5ef4e2a7fdd1154128a826c00b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <794884194914d03c4e199d9c1f090b0c> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <2d8a25919ea488ce008d3635d9b104c7> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <d720d27ae2c51977cf9ea895f5ed6c00> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <12cb1ed29943213839f0b0d18cd42761> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <cf3795bdb521e4fc8a67ec1b956cc1fd> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <110d7f3159c1d306851c3ec5c6d302ef> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <2b086ebf6fdb359efec30f0fe50a4b5e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <73400f494c8d580bd774443a5163127b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ca273d664d1a6c59a5a002670a641eff> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <82e03120eb2e393fdb39f4a2408e14c9> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <de4e56af446ae2064ab112c46e73ceba> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <6862f26ba0bd11ecb058676e99192762> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9d8f681f65528f50688f369bfbe31966> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <8e77edbc0a107e2240b8952cd97dfe49> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <197f4ab2ce1ac63b484abba01db0315d> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9fb771b90472d6e0ccaf1bb880c530ca> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <44213278b52d888683e970004dc95f3c> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <a5ce2b0dd7e4ecb7057936890feefbbe> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <6bb7eee188f156a5bafe99bbbcd77be9> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <b0b16feb87b9491a8f14784ac728e2d7> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <e018b09caa2c4e63ab143394a9d68d74> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <4418c3e08fa5b60d93f0f1dd30b66815> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <a242915187d12f5b6d5072165fba4b5d> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <95ca807f05d258f7723d5f1f75c29159> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <8058312284b1223b4e9fe8b096e487c9> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ffc8d3472451090677fd446837e384ff> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <2fb8b45523a7aed73e80973854aaa9ec> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <7fd8e8e400f9c6c9e54e04629a0ac1d4> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <192eed7af8e3311445372f2a43cb63ec> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f3a749e3b0d0e4d27a4b30a57b365911> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <1f163087a2d335a87bd8dd4ceaf162f2> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <090b8eda3ecfcbe44c68a657dc62a67b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <36a7b9a76f1df2a339ba619568646265> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <219a60c23a74cf1ede2ee1c56dc790d7> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9732cf2c564fa6494016d5070296dc72> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ebe207986fba436c1de3f43adb0ea91d> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ba2780f844f711525924923e9bfb23cb> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <4bbc616d609e155e2183ebf7cdb51ed0> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9390d128a161188c0866913e588af143> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c62644119a8eb95abbe4647ead099d03> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <1ba3f953b8e28bd2f5a5ec2e741edf65> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <bce68ef38c4919e28cb2c31ac07fbd01> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <3352c4e7ef807c876a7f5de2964df8d5> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <6ee8d89ae74833c397169c07b81e62e2> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <1e8b9f9cf868b915cc06ad6eaf7e280b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <58cea53bfb32988291b49a6205388cd1> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c03bcc5418d9a06050e23c7daa68d5b2> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <37c6058f80a07e9becba642495e74d2f> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <3ef877eea930bd29355abe662191c2c6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <b5827d19822c0668babd9a567e353930> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <3880f6fe422f366324130a8be948581d> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <fbdd5fe680fc061a5ff4cdba3c681e27> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <5f83691039dd4abf560de0861a248faf> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <d7881baf17ece4f8683923d9b1df6f48> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9182603972210f87c7fbccfa4299b62d> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <7e9ba202e5f83fe3b453239e01e3f013> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f93cf31e51dcf68add465b5690421c42> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <46f2cd89435b46acdead93ac25fc38a3> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <576e117a4c895757165f6071ecf836b6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <671043fee11dff26c1d56732e1957a04> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <a75c35ae3687a0abcc224fafafe44618> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c627488e2e6d3506d85a85e3fb4447f5> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f75a1ef4388e2132d8b8c7527ce6b85e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <57633f0bb4c60915c13f78ef4f433b72> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9d314cde286ee00063f70980126cacff> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <519785fdc0f19ac89befaefbb015f6f7> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <3a9c2c862b4b57eefbc107426a1f9126> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <cde9774cd719ccb6b9c9101316665aa6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <243622fc2f2772f05b94e34af73c355e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <3199bdcac04ba64f6ce1a3a2109ef9e3> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <545467bd5d480e93f278fbf9bc2fb61e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <421a76b3c3c2236b06b427de482ab850> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <19b02ed8707958f798d02437f5564f58> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c1c04c5dfdb2d647a0a5e6bf54bc4b07> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <42f9397c4362faa9cf62ac6da6a41f5a> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ae2b88c417c7d2ebfe3390535643ce8c> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <e332613f072168f9f84be81cccde18cc> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <e74cac00dbb09264098923f13f1b29bd> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <1fee067306e8dc5f31b7020de3f0e1d6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <fc2eefe0712efdd627716c2bd4d9d982> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <e80bd281bd26e00d735aada7b7b94c7a> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f22261659c0c4f4f0ab7bf6f4faa6323> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <e251b297ac85e7e349de4a6af04df637> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f6e2cfb8779eaf43a78b53381138416b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <e916365766702b768c93c28dd0caf168> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c684e0ce02bb2a757116a43bf2b277ec> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <2649c18cedba042ff743f45af76b7e5e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <0e344739ef5187a307f6b63fcb19d1db> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ff60322849b6081162dafc46536ce5ab> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <94709e68c8b82c871f6bf3e8bf6dd229> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <eb33ae26edd21211407b9b665dd9b05e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <54b3b8de8a43f5c7351649c137172f3e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <bd4cdddb5102ab425626d83e9c92e03b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <cc53aa08b339a411f07d8590a246258d> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <2cdeac649f30cedacc484820aa1dd3dd> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <86bdc14eb42917dc0de3d366e9549643> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <af1ec9121da534836e6a39b7d261fa65> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <bfca5ea1eb6a28d5f7a20afe252f41cf> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9455c4da654254f8b15e0910c42f8be8> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <4515ff927df3aa27fb4a9fc07b3a74e9> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f01d55d1df072d62ab79ac6e9ff82a62> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <295340357a700b8f3653ac9085c88b5e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <0c2b237422ed96c2acc7153f99e767b3> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <6ad073b593b69bc9bd5d299047ed0d7e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <4fe37a4e22f312f89f23f50d0ae30185> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <e5ca53ad41889be06a595f36db5b3c26> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <a430b813dd5c253002380cda46bf8a05> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9c5ea1c2659533cddf5e3d76e8a69546> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f4afb44c0c7b701e8d23c4dbb4de36c2> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <2a2c53858ac1ef449ff10c402a5e512c> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c3ee1205e9bb8b0378ca08e1eb2d90f9> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <2eabad04f9145d32a6a3eda285d66c39> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <71a95b26dacd628c80b1d7255e983b33> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <4dd9ed5f1d35aa5e95f9e34477bef74d> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f714a1debfb5a7c8f74cc9c81fc0a137> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <6ba4df8ec7f814fc963347d9ee2a4b8e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <d251094d62d13ad5e04652138e39b2e2> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <833ac927093e8d33370bed3655844587> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <d2f35c24a9bfcdbe4634363e57fbeffe> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9d950acda37040a5e1eed09345a5ebf2> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <383c8351d41296285b58708b8b23373a> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <eb0f6061d9d1caef5d0010b29e05a239> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <2507f5be22dc8e10252c641e191bef4a> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <7f308eb80e075166d79bc5c453b1a1f4> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <3b171e0b79acf468ae1bf3a6d8515d12> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <32dc0bee1ddee3de7e74fab40a124e7b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <0db1c69d0a21974d88449c873f2b16b6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <b2c4e4c540dcd2fd05dc48c165122882> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c79149286d07d2b2be3f167405eedda6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c5456ca67f7f849bf657ed23b6b2ce90> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <bbc29395549d0fcb7b4f5540c5625015> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <b4ac9705550f189cf2aff4d4748bd05e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ef7b149d4b5169d6509be8b1edd58427> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <6f1df4f453b5e960a7850b434a76693c> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <a21f547bc1f70043a28c4e2d5b04e570> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <757e78163b7274540efd4a0838adf7e0> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <8e5ed17e3bd0c551774c74053cf2bd8b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <2461beb425affab106efe8f01734e5fa> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <8c5f7a7c73bc00247874fefe6b4a1d70> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <92eb1676b68fa0c9de9a9b7dc1a187c8> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <d3d32aadf4fd75f90770348cbdf1ddfd> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <d5d6cdb6d43cea76db89a50c6a7c3bcf> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <cbe237b1c7b5f2ff6cf2c505ff49c730> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <b2187b4af3c4cba81b00fcf8bc7897de> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <053db2ffc4105a601f6ac5f23cc356d6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c508bc43c9f2ae2523be36a9d8fd5bc6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ce32ad052da5f355e968cfae2ecc1cea> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <00b6a1fdbc5b53afac5f9b056349f7cc> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <72173e4ac13a8ba1ed47a1522aae8a14> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f11372a5e8cc737755143bf00a65a857> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <17c5ca481ee2f4992ce479881b07b6aa> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <3f94d68f3f0eef8fdd0648ba3da43ef6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <0921f97ae9a04d0d0fc22983888bbe7d> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c3ab2eb136ce67f1f0b0346cb7f9e2ff> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <962d3065fdd106bffa3954089e328e47> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f258b0151578b61520574237c1eaa1a8> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <6b3ef56fd0622a44204b0c2bf014df70> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <456a0c7be5d912e51e23183edc77d39a> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ea789dda17a7324f83daed52dc1dd064> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <0041c70c01f9fb4c88acf6eb0675261f> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <d46ce3e8ed95465cfb65d36622f603d6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <719ca3d7a172f86f6bb44c8fcd068bc3> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f97045b682cc350eb342e6a58f5b4b94> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f9f62880839e477c535aef7231ef2ebe> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <23dfe3ab2b2523179a9fe1f22c18c13e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <1d66b11801aa2376ac489ea45371575e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9d96b8237e4855e41af5819a06212381> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <15b14a32d2d7a2f7dc60000f7ee91875> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <16d0a1ccf44ba9f0e529d90765992bc7> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <a5831c2b239ebfc6f13db177abd346dd> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <71ffc6df68662d35959169b71a55468e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <0e9df50566390f7da1a180806ea7459a> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <16f5b182da0953e5f54d1df76fbc4a10> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ef98bc2981f7d1e793bb391ca0208231> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <6ea9ba935598917da4dd305962c2760c> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f0921573abf41194ce99f3976f3c7792> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f8a4bdb249dbe18c83e254b52edad748> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ba209ba3b0d3bce1a99fc412d3b5c81f> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <e14acc86fb97dc994ccb55386f3acb52> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <1f1f6b03483b52744440fba1261d3e8f> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <5301c222f995f460d6fccfbfb1f6c933> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c1c4d4deec691ad364853ff6c06879ff> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <b0e4fd968fb5ec319e76d250093f54d6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <689d8066bc9c21074bce00b1357ec584> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <0d0b05a7bc59b1ef41e506a49a867f63> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <276f135daae8851a13781207a10ffe39> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <7bc02760008fece9c6c82c7076b2084b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ff4d35444e8b7976ced7be2db7812614> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9ed482b54aaf5e2732cb7d90f42f40c1> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c1350ada18fb781c59eb657e5beca680> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <b8582f9bd2ab4006f56a49dcfc382b86> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <31d3f96d1ffb5f47676fdc57cebe7d5b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <604b95adfa625746d1c9e0c18d7cef75> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <11796bb40d515104e7e7d9f37757869e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c6371115c38d9164892ebdaa5bbec4b2> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <822154824ef52d7ad048744ba1f8c7cf> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <4bbb5ceb0c3182c1680a7441de3ec2ab> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <853f54b85e789b9994e1a5e91b626df6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <7e054b7b7d705b1db32c900cd6e861f8> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <1763467b83434ea9385ecb53807bbc0e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <85ab2b1b8e9c9961c5226fc1fcb707b3> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9e21f49afdaae8885663e807606387a2> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <0c8b65a907c7d0c6fa14cc628ece64ad> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <efe14db04214c06033ca218b0e4b29e4> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <0f5a6232102792b2e98c19209070d4f8> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ae0bf1158fa4fc032ac6dd310bdd89fe> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <a96f3bb4a7b2778841cc862a210f5394> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <7f099559e207a4ffb9d118cce75cfbb6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <a4eb0ef04cef1a31c57d1e33eb303ef0> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c266fa77af262ed1cde8bc66b2f716a3> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <6e99e1b61e564361a76af71557904977> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ac31bf081f355d8ffbb73f0279341474> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <e06d049499494a1234d50bef8d68ee75> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <79a01afd24caa341f685dba77dd2e337> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <21132ce79197b4b12857b809012cd28b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <45d92a75dc8d02e7ae8ae006853ad27a> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <32d1bbadde2004d3783dd9e4361b30c4> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <884e576d2b0b4ccd8990d51f8c554ef0> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <b4cadf9552194c3fc6096bd42dea35f9> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <91a9a1c3248a72259cabfab005b75a15> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <7a9f13b6fe0a6a2b25e6807a95766c3b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <3b528f335fe9472004f43422c5016bd3> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <d91a4e029d62fc64ca7d8f54f52efd11> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <71f67e5defd2743e2507c148945308e8> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <d454682cc53e9fa59b18c8d35ba62b18> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <b30071c260ebf810defaad28092b9750> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <d4c811a40ed73a2c8b892d7d48e8dc3a> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <bd5f6e706ff2996c4c686eb2a6796dec> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <572fe1aa6525c8b31107aaaec0d9fa03> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f3db693b152ca34f58191d52e7bed2c2> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <611ed890ce0b13ab0a581563ffd044c0> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <32da31ddce3388353cf437708e08f4e6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <a11882722af2dd1f5f2dd7903d8728d2> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <2d17582b557402f60d4013de3e5d18a2> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c5e5953f47efccaa93ae8eb503096e64> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <cb470c16cb920e1a766174d4a8a1af25> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c6923743983924061686da9a2061dcb4> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <7ce5dcd4aba5f7f2c441797bddaa811b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <e1a8a3a6644ef291bf0543bac268498d> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9c9c4f6e7cb1c81b9094a3a07850594a> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f2406f55c853b6138b3296c2d7763cd8> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <893609eb1317367ce3986f04b95973d9> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <fe6dc57d2ffecdfcfeff25d8469659db> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <2d9aafcd6ac46b5c3eb5dc931c6c954c> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <64c53ef36e49777814b3272260722238> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ee510a0011615b50d2179be248503952> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <2c7cdd4d71602b9f12e0e8d641afb043> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ccd2ac1fd2765c5b83dfef6a7152e40a> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c4de028671f01d9649aabb85e73b50a4> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <bf6e42603aec4e28a5b1dd34f6509e48> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <cd2e88fd365dcced44ef3e1baf27baa5> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <47850a3b99243cfb1147e8856bab2691> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <65f6ae3fdb9ccefbc2a6d0da6653e11a> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ea4a5a9a01c9ad6e39718716e6cf9c06> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <759839959bfb6316fce28083afd9f5e5> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f4f5727a2137b862a363dc96bb53e982> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <59d8be5135a380156fb24316c40c7bbb> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <37aa92969b9a1af58c7aa1ebaae855c6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <7a66347317dc870d4892444a1675b668> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <d7397c1ae5f83017023c5f2824f71e37> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <55b9d95b276a45acd31d3f18f8c5a9b6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <a4a5e861626ce97c9aa544d9ecdf6d22> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <dae0e3890d4f47ff0821afe36eb5c369> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f4f0dd21c715f08290fbea884c56a0e6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <645b9f5e3f75c6794d1497aba54e41b2> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <2f7f25d16d5d16dece60f5fbe796d545> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <42464efb1d4fe2615c1016e24c3a7c86> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9b38bf4e83f6272a293affa8c358827b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <4447b200737e3fead7d054e5fc58e081> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <e33825a95f4bf5e920552dec305cd9f7> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <8b21200caa4ea9dda748ffb5d12737cf> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <b796955f5ef84ea2c5837bb36009b343> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <004bac27bcba1d4fb5540d8c78b6f73e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <e7147ab3de27b16b9e7dda610244dcf7> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <664020dc1c08b8e03c17fe09bc5627b8> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <187055aafea8f2877dbd1ac99e417d37> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <4a890e0e7c0cc69df96c7d4db8992341> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <34157b152e71ee3bba3ca1d380534062> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <1a06916757babba002fcc87cb5470c92> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <667facd0fd90a563e0267bf3b4778125> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9ec8b4147d69efb4318d09076db276b0> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <87ae44119088cf743809b524ee5be96b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <062434d50407e18a5123520e5e9167f3> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <481cdb54073392c2e729011a798bad75> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <bfcac41a298e2b192d2766ec93a1ada3> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c7b561328faf7c53734fca0ac3684263> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <1dcdb7ebc4cd572e3775983d00cd8882> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f3a760f35cae6a6c0571f6cb12fb3093> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <d53a8b99786f619d4c36758b2caf33f2> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <7e794222094a3247605b6b1925fbf731> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <3eb8ca62221d66c2c86007581e7d7f1a> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <7f398290ce078ddb840df11c3c6980b2> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <df767a875f8e46ee28e6e5ae127bef78> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <8954e3fe4558250315b78b2bbfc2c041> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <cd7115076ec817bd1b053f10d5662f24> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <420e5f61e2497c9b6ba81c13b0064d3f> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <0195751ebfafcb94b170b2ef76d76d0d> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <8758461c242121b1fbbb1a540d3bc238> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <e7fca65962dab4f76efda3c3a9bf7fc6> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <437c58747a7cbfadd3425540b6047317> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9449c98f67cb91198317f1fad83cc49e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <70d7dd713f24b790c8589542ba2ff82d> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <83ad2cf23359fa8987ba614e725fd7c1> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <19fa0a9fca96c4f11663349add955f11> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <37617a9e1a4f24d4404df7ca50ca815b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <db3ca61c7db9e818751d2c4d4c1bca5e> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c8bc4c6a64feac06065845cfb94899dd> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <294b765dbef1146de43dd3ae7c60101a> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <04ca4a7eee1478f26f92e5a5df37c703> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <dae8a43601ff2c6795d9d5ae60624340> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <4359fe88ba37970ce284ae6214354f91> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <9dc62732fb5588ad4b2a3e3e8da5f346> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <4c165be9310bc7680448932fde9ec2bc> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <f83661aed7c7dd80e1dbbef00dc55ad9> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <71beffd4ecd747929eefd4fef5133856> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <d0eeb3b13e605cabd855d393d9555070> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <09d6ba103ad4c52bc4212210ffcaf35f> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <b8e6ec9fa0f4626cac15e66952d7e627> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <d251bfd8e30540b5bd219bbbfcc3afc5> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <e3dde0e2bc07789ddb90b3558dd158cc> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <63c1c0678ab1e45835ff2ae62f94620f> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <ec13b7a5a79144a287d8a5490aaff0ec> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <bbb0064abb59c204a9dd4ebb84ea0591> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <267a56c503973426271080ebfa31299b> from storage
[2026-04-13 18:10:13] [Debug]    Loaded path table entry for <c595e4fda00a87ec51ae26c63bfe6d43> from storage
[2026-04-13 18:10:13] [Verbose]  Loaded 346 path table entries from storage
[2026-04-13 18:10:13] [Verbose]  Loaded 0 tunnel table entries from storage
[2026-04-13 18:10:13] [Notice]   Transport Instance will respond to probe requests on <rnstransport.probe.e41c0bc1e2b5704f328bae9b3dbc0839:d7fa9b80db0ecbffe303616bebe78f74>
[2026-04-13 18:10:13] [Verbose]  Transport instance <e41c0bc1e2b5704f328bae9b3dbc0839> started
[2026-04-13 18:10:13] [Notice]   Started rnsd version 1.1.5
[2026-04-13 18:10:14] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:14] [Debug]    Replacing destination table entry for <110d7f3159c1d306851c3ec5c6d302ef> with new announce, since it was more recently emitted
[2026-04-13 18:10:14] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:14] [Extra]    Valid announce for <8f7eb4779cd55038497fda243f094a31> 11 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:14] [Debug]    Destination <8f7eb4779cd55038497fda243f094a31> is now 11 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:14] [Extra]    Valid announce for <11adcf54ca64d4a6017c8911440d6b82> 37 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:14] [Debug]    Destination <11adcf54ca64d4a6017c8911440d6b82> is now 37 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:15] [Debug]    Rebroadcasting announce for <110d7f3159c1d306851c3ec5c6d302ef> with hop count 5
[2026-04-13 18:10:15] [Debug]    Rebroadcasting announce for <8f7eb4779cd55038497fda243f094a31> with hop count 11
[2026-04-13 18:10:15] [Debug]    Rebroadcasting announce for <11adcf54ca64d4a6017c8911440d6b82> with hop count 37
[2026-04-13 18:10:15] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.25ms
[2026-04-13 18:10:15] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.98ms
[2026-04-13 18:10:15] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.76ms
[2026-04-13 18:10:15] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] for processing in 6.64ms
[2026-04-13 18:10:15] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.5ms
[2026-04-13 18:10:15] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.48ms
[2026-04-13 18:10:15] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.46ms
[2026-04-13 18:10:15] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] for processing in 6.44ms
[2026-04-13 18:10:16] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:16] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:16] [Extra]    Valid announce for <95ca807f05d258f7723d5f1f75c29159> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:16] [Debug]    Destination <95ca807f05d258f7723d5f1f75c29159> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:16] [Extra]    Valid announce for <13da3e301bfb15dc0d0499859e0e7cfe> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:16] [Debug]    Destination <13da3e301bfb15dc0d0499859e0e7cfe> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:17] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:17] [Extra]    Valid announce for <13da3e301bfb15dc0d0499859e0e7cfe> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:17] [Extra]    Heard a rebroadcast of announce for <13da3e301bfb15dc0d0499859e0e7cfe> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:17] [Extra]    Valid announce for <93f793207919f56ff52449bcf41b244e> 9 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:17] [Debug]    Destination <93f793207919f56ff52449bcf41b244e> is now 9 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:17] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 4
[2026-04-13 18:10:17] [Debug]    Rebroadcasting announce for <95ca807f05d258f7723d5f1f75c29159> with hop count 3
[2026-04-13 18:10:17] [Debug]    Rebroadcasting announce for <13da3e301bfb15dc0d0499859e0e7cfe> with hop count 2
[2026-04-13 18:10:17] [Debug]    Rebroadcasting announce for <93f793207919f56ff52449bcf41b244e> with hop count 9
[2026-04-13 18:10:17] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.07ms
[2026-04-13 18:10:17] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.89ms
[2026-04-13 18:10:17] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.79ms
[2026-04-13 18:10:17] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] for processing in 6.67ms
[2026-04-13 18:10:17] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.53ms
[2026-04-13 18:10:17] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.51ms
[2026-04-13 18:10:17] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.5ms
[2026-04-13 18:10:17] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] for processing in 6.48ms
[2026-04-13 18:10:17] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.46ms
[2026-04-13 18:10:17] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.44ms
[2026-04-13 18:10:17] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.43ms
[2026-04-13 18:10:17] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] for processing in 6.42ms
[2026-04-13 18:10:17] [Extra]    Valid announce for <c09a5cf4c1ef615844eeb86ea928b47b> 12 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:17] [Debug]    Destination <c09a5cf4c1ef615844eeb86ea928b47b> is now 12 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:17] [Extra]    Valid announce for <93f793207919f56ff52449bcf41b244e> 9 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:10:18] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:18] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:10:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:18] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-13 18:10:18] [Warning]  Attempting to reconnect...
[2026-04-13 18:10:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 18:10:18] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:18] [Debug]    Rebroadcasting announce for <c09a5cf4c1ef615844eeb86ea928b47b> with hop count 12
[2026-04-13 18:10:19] [Extra]    Valid announce for <4cce8a55cc0f232fb0946b392a73fa92> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:19] [Debug]    Destination <4cce8a55cc0f232fb0946b392a73fa92> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:19] [Debug]    Path request for <60dcab5ef4e2a7fdd1154128a826c00b> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:19] [Debug]    Not answering path request for <60dcab5ef4e2a7fdd1154128a826c00b> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], since next hop is the requestor
[2026-04-13 18:10:19] [Debug]    Rebroadcasting announce for <4cce8a55cc0f232fb0946b392a73fa92> with hop count 3
[2026-04-13 18:10:19] [Extra]    Valid announce for <4cce8a55cc0f232fb0946b392a73fa92> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:19] [Extra]    Valid announce for <ffc8d3472451090677fd446837e384ff> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:19] [Debug]    Replacing destination table entry for <ffc8d3472451090677fd446837e384ff> with new announce, since it was more recently emitted
[2026-04-13 18:10:19] [Debug]    Destination <ffc8d3472451090677fd446837e384ff> is now 6 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:20] [Extra]    Valid announce for <4ba30ddbc5e3639be796a98117bb08a8> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:20] [Debug]    Destination <4ba30ddbc5e3639be796a98117bb08a8> is now 5 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:20] [Extra]    Valid announce for <4ba30ddbc5e3639be796a98117bb08a8> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:20] [Extra]    Valid announce for <0644032081ec4c00a0114c648214de30> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:20] [Debug]    Destination <0644032081ec4c00a0114c648214de30> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:20] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:20] [Debug]    Replacing destination table entry for <e345f6220682e127cab52c3387436778> with new announce, since it was more recently emitted
[2026-04-13 18:10:20] [Debug]    Destination <e345f6220682e127cab52c3387436778> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:20] [Extra]    Valid announce for <053db2ffc4105a601f6ac5f23cc356d6> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:20] [Debug]    Destination <053db2ffc4105a601f6ac5f23cc356d6> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:20] [Debug]    Rebroadcasting announce for <ffc8d3472451090677fd446837e384ff> with hop count 6
[2026-04-13 18:10:20] [Debug]    Rebroadcasting announce for <0644032081ec4c00a0114c648214de30> with hop count 3
[2026-04-13 18:10:20] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.8ms
[2026-04-13 18:10:20] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.45ms
[2026-04-13 18:10:20] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.32ms
[2026-04-13 18:10:21] [Extra]    Valid announce for <d62f88565958b2cd1045cf7f74796ce8> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:21] [Debug]    Destination <d62f88565958b2cd1045cf7f74796ce8> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:21] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:21] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:21] [Extra]    Valid announce for <bb3819350c3c8ddaea5c441868e1699c> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:21] [Debug]    Destination <bb3819350c3c8ddaea5c441868e1699c> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:21] [Extra]    Valid announce for <d62f88565958b2cd1045cf7f74796ce8> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:21] [Debug]    Rebroadcasting announce for <110d7f3159c1d306851c3ec5c6d302ef> with hop count 5
[2026-04-13 18:10:21] [Debug]    Rebroadcasting announce for <8f7eb4779cd55038497fda243f094a31> with hop count 11
[2026-04-13 18:10:21] [Debug]    Rebroadcasting announce for <11adcf54ca64d4a6017c8911440d6b82> with hop count 37
[2026-04-13 18:10:21] [Debug]    Rebroadcasting announce for <4ba30ddbc5e3639be796a98117bb08a8> with hop count 5
[2026-04-13 18:10:21] [Debug]    Rebroadcasting announce for <e345f6220682e127cab52c3387436778> with hop count 5
[2026-04-13 18:10:21] [Debug]    Rebroadcasting announce for <053db2ffc4105a601f6ac5f23cc356d6> with hop count 4
[2026-04-13 18:10:21] [Debug]    Rebroadcasting announce for <d62f88565958b2cd1045cf7f74796ce8> with hop count 4
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.47ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.38ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.33ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.27ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.27ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.26ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.25ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.24ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.24ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.23ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.22ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.22ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.21ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.2ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.2ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.19ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.18ms
[2026-04-13 18:10:21] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.18ms
[2026-04-13 18:10:22] [Extra]    Valid announce for <bb3819350c3c8ddaea5c441868e1699c> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:22] [Extra]    Valid announce for <d20a381e1b46d0855c46105b565ec8ce> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:22] [Debug]    Destination <d20a381e1b46d0855c46105b565ec8ce> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:22] [Extra]    Valid announce for <e59c163a01cac4628dace75d8a8c5efe> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:22] [Debug]    Destination <e59c163a01cac4628dace75d8a8c5efe> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:22] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:22] [Debug]    Replacing destination table entry for <02aaf088472435718061211d3752c8ed> with new announce, since it was more recently emitted
[2026-04-13 18:10:22] [Debug]    Destination <02aaf088472435718061211d3752c8ed> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:22] [Extra]    Valid announce for <1f163087a2d335a87bd8dd4ceaf162f2> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:22] [Extra]    Remembering ratchet <2b23ec15c571cde781a3> for <1f163087a2d335a87bd8dd4ceaf162f2>
[2026-04-13 18:10:22] [Debug]    Replacing destination table entry for <1f163087a2d335a87bd8dd4ceaf162f2> with new announce, since it was more recently emitted
[2026-04-13 18:10:22] [Debug]    Destination <1f163087a2d335a87bd8dd4ceaf162f2> is now 6 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:22] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:22] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:22] [Extra]    Valid announce for <d20a381e1b46d0855c46105b565ec8ce> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:22] [Extra]    Valid announce for <e59c163a01cac4628dace75d8a8c5efe> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:22] [Extra]    Completed announce processing for <110d7f3159c1d306851c3ec5c6d302ef>, local rebroadcast limit reached
[2026-04-13 18:10:22] [Extra]    Completed announce processing for <8f7eb4779cd55038497fda243f094a31>, local rebroadcast limit reached
[2026-04-13 18:10:22] [Extra]    Completed announce processing for <11adcf54ca64d4a6017c8911440d6b82>, local rebroadcast limit reached
[2026-04-13 18:10:22] [Debug]    Rebroadcasting announce for <ca273d664d1a6c59a5a002670a641eff> with hop count 3
[2026-04-13 18:10:22] [Debug]    Rebroadcasting announce for <bb3819350c3c8ddaea5c441868e1699c> with hop count 4
[2026-04-13 18:10:22] [Debug]    Rebroadcasting announce for <d20a381e1b46d0855c46105b565ec8ce> with hop count 4
[2026-04-13 18:10:22] [Debug]    Rebroadcasting announce for <e59c163a01cac4628dace75d8a8c5efe> with hop count 4
[2026-04-13 18:10:22] [Debug]    Rebroadcasting announce for <02aaf088472435718061211d3752c8ed> with hop count 5
[2026-04-13 18:10:22] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 11.23ms
[2026-04-13 18:10:22] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 11.0ms
[2026-04-13 18:10:22] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 10.82ms
[2026-04-13 18:10:22] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 10.65ms
[2026-04-13 18:10:22] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 10.62ms
[2026-04-13 18:10:22] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 10.6ms
[2026-04-13 18:10:22] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 10.56ms
[2026-04-13 18:10:22] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 10.53ms
[2026-04-13 18:10:22] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 10.52ms
[2026-04-13 18:10:22] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 10.48ms
[2026-04-13 18:10:22] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 10.46ms
[2026-04-13 18:10:22] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 10.44ms
[2026-04-13 18:10:23] [Extra]    Valid announce for <9f3a20b50352a2523193caa1b5130f5d> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:23] [Debug]    Destination <9f3a20b50352a2523193caa1b5130f5d> is now 5 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:23] [Extra]    Valid announce for <95cb09df57c070a2e9c6a35831e9904c> 10 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:23] [Debug]    Destination <95cb09df57c070a2e9c6a35831e9904c> is now 10 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:23] [Extra]    Valid announce for <c077d6b4bf557e71faebf52aa54a74d2> 9 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:23] [Debug]    Destination <c077d6b4bf557e71faebf52aa54a74d2> is now 9 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:10:23] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:23] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:10:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 18:10:23] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:23] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-13 18:10:23] [Warning]  Attempting to reconnect...
[2026-04-13 18:10:23] [Extra]    Valid announce for <219a60c23a74cf1ede2ee1c56dc790d7> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:23] [Extra]    Remembering ratchet <e0e56b8405eb1e30cc81> for <219a60c23a74cf1ede2ee1c56dc790d7>
[2026-04-13 18:10:23] [Debug]    Destination <219a60c23a74cf1ede2ee1c56dc790d7> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:23] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 4
[2026-04-13 18:10:23] [Debug]    Rebroadcasting announce for <95ca807f05d258f7723d5f1f75c29159> with hop count 3
[2026-04-13 18:10:23] [Debug]    Rebroadcasting announce for <13da3e301bfb15dc0d0499859e0e7cfe> with hop count 2
[2026-04-13 18:10:23] [Debug]    Rebroadcasting announce for <93f793207919f56ff52449bcf41b244e> with hop count 9
[2026-04-13 18:10:23] [Debug]    Rebroadcasting announce for <1f163087a2d335a87bd8dd4ceaf162f2> with hop count 6
[2026-04-13 18:10:23] [Debug]    Rebroadcasting announce for <73400f494c8d580bd774443a5163127b> with hop count 3
[2026-04-13 18:10:23] [Debug]    Rebroadcasting announce for <9f3a20b50352a2523193caa1b5130f5d> with hop count 5
[2026-04-13 18:10:23] [Debug]    Rebroadcasting announce for <95cb09df57c070a2e9c6a35831e9904c> with hop count 10
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.07ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.82ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.67ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.53ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.51ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.48ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.43ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.26ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.24ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.19ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.16ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.14ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.09ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.07ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.04ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.0ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.97ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.94ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.9ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.87ms
[2026-04-13 18:10:23] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.85ms
[2026-04-13 18:10:24] [Extra]    Valid announce for <219a60c23a74cf1ede2ee1c56dc790d7> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:24] [Extra]    Heard a rebroadcast of announce for <219a60c23a74cf1ede2ee1c56dc790d7> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:24] [Extra]    Valid announce for <c077d6b4bf557e71faebf52aa54a74d2> 11 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:24] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:24] [Extra]    Remembering ratchet <0a0db4208f06e28af0c1> for <794884194914d03c4e199d9c1f090b0c>
[2026-04-13 18:10:24] [Debug]    Destination <794884194914d03c4e199d9c1f090b0c> is now 6 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:24] [Extra]    Completed announce processing for <2d8a25919ea488ce008d3635d9b104c7>, local rebroadcast limit reached
[2026-04-13 18:10:24] [Extra]    Completed announce processing for <95ca807f05d258f7723d5f1f75c29159>, local rebroadcast limit reached
[2026-04-13 18:10:24] [Extra]    Completed announce processing for <13da3e301bfb15dc0d0499859e0e7cfe>, local rebroadcast limit reached
[2026-04-13 18:10:24] [Extra]    Completed announce processing for <93f793207919f56ff52449bcf41b244e>, local rebroadcast limit reached
[2026-04-13 18:10:24] [Debug]    Rebroadcasting announce for <c09a5cf4c1ef615844eeb86ea928b47b> with hop count 12
[2026-04-13 18:10:24] [Debug]    Rebroadcasting announce for <c077d6b4bf557e71faebf52aa54a74d2> with hop count 9
[2026-04-13 18:10:24] [Debug]    Rebroadcasting announce for <219a60c23a74cf1ede2ee1c56dc790d7> with hop count 3
[2026-04-13 18:10:24] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.6ms
[2026-04-13 18:10:24] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.19ms
[2026-04-13 18:10:24] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.91ms
[2026-04-13 18:10:24] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.64ms
[2026-04-13 18:10:24] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.59ms
[2026-04-13 18:10:24] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.56ms
[2026-04-13 18:10:25] [Extra]    Valid announce for <13da3e301bfb15dc0d0499859e0e7cfe> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:25] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:25] [Debug]    Replacing destination table entry for <2d8a25919ea488ce008d3635d9b104c7> with new announce, since it was more recently emitted
[2026-04-13 18:10:25] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 6 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:25] [Extra]    Completed announce processing for <c09a5cf4c1ef615844eeb86ea928b47b>, local rebroadcast limit reached
[2026-04-13 18:10:25] [Debug]    Rebroadcasting announce for <4cce8a55cc0f232fb0946b392a73fa92> with hop count 3
[2026-04-13 18:10:25] [Debug]    Rebroadcasting announce for <794884194914d03c4e199d9c1f090b0c> with hop count 6
[2026-04-13 18:10:25] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.15ms
[2026-04-13 18:10:25] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.67ms
[2026-04-13 18:10:25] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.24ms
[2026-04-13 18:10:26] [Extra]    Valid announce for <c077d6b4bf557e71faebf52aa54a74d2> 13 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:26] [Extra]    Valid announce for <13da3e301bfb15dc0d0499859e0e7cfe> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:26] [Extra]    Completed announce processing for <4cce8a55cc0f232fb0946b392a73fa92>, local rebroadcast limit reached
[2026-04-13 18:10:26] [Debug]    Rebroadcasting announce for <ffc8d3472451090677fd446837e384ff> with hop count 6
[2026-04-13 18:10:26] [Debug]    Rebroadcasting announce for <0644032081ec4c00a0114c648214de30> with hop count 3
[2026-04-13 18:10:26] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 6
[2026-04-13 18:10:26] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.93ms
[2026-04-13 18:10:26] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.7ms
[2026-04-13 18:10:26] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.64ms
[2026-04-13 18:10:26] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.58ms
[2026-04-13 18:10:26] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.53ms
[2026-04-13 18:10:26] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.52ms
[2026-04-13 18:10:27] [Extra]    Valid announce for <020a777c04438f287caa9df8c6e99834> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Debug]    Destination <020a777c04438f287caa9df8c6e99834> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <985c7a6f4ed73219006b92b30b8b282c> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Debug]    Destination <985c7a6f4ed73219006b92b30b8b282c> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <4cce8a55cc0f232fb0946b392a73fa92> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Debug]    Destination <192eed7af8e3311445372f2a43cb63ec> is now 2 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <9c4780953ea0297e80cc1797ffe36deb> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Remembering ratchet <6dfd0b5696a71b6c3f2d> for <9c4780953ea0297e80cc1797ffe36deb>
[2026-04-13 18:10:27] [Debug]    Destination <9c4780953ea0297e80cc1797ffe36deb> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <8cdee9fb325052d66800632d7d3773b0> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Debug]    Destination <8cdee9fb325052d66800632d7d3773b0> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Heard a rebroadcast of announce for <73400f494c8d580bd774443a5163127b> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <053db2ffc4105a601f6ac5f23cc356d6> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Heard a rebroadcast of announce for <ca273d664d1a6c59a5a002670a641eff> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <44213278b52d888683e970004dc95f3c> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Debug]    Destination <44213278b52d888683e970004dc95f3c> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <b993f1def2a655d9ef7b723f14a4d7b4> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:27] [Debug]    Destination <b993f1def2a655d9ef7b723f14a4d7b4> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <95ca807f05d258f7723d5f1f75c29159> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <13da3e301bfb15dc0d0499859e0e7cfe> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <93f793207919f56ff52449bcf41b244e> 9 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <ffc8d3472451090677fd446837e384ff> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <0644032081ec4c00a0114c648214de30> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Heard a rebroadcast of announce for <0644032081ec4c00a0114c648214de30> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <4ba30ddbc5e3639be796a98117bb08a8> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Heard a rebroadcast of announce for <4ba30ddbc5e3639be796a98117bb08a8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <d62f88565958b2cd1045cf7f74796ce8> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <bb3819350c3c8ddaea5c441868e1699c> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <b993f1def2a655d9ef7b723f14a4d7b4> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <ba2780f844f711525924923e9bfb23cb> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:27] [Debug]    Replacing destination table entry for <ba2780f844f711525924923e9bfb23cb> with new announce, since it was more recently emitted
[2026-04-13 18:10:27] [Debug]    Destination <ba2780f844f711525924923e9bfb23cb> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:27] [Extra]    Valid announce for <c62644119a8eb95abbe4647ead099d03> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:27] [Debug]    Destination <c62644119a8eb95abbe4647ead099d03> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:27] [Extra]    Completed announce processing for <ffc8d3472451090677fd446837e384ff>, local rebroadcast limit reached
[2026-04-13 18:10:27] [Debug]    Rebroadcasting announce for <4ba30ddbc5e3639be796a98117bb08a8> with hop count 5
[2026-04-13 18:10:27] [Extra]    Completed announce processing for <0644032081ec4c00a0114c648214de30>, local rebroadcast limit reached
[2026-04-13 18:10:27] [Debug]    Rebroadcasting announce for <e345f6220682e127cab52c3387436778> with hop count 5
[2026-04-13 18:10:27] [Debug]    Rebroadcasting announce for <053db2ffc4105a601f6ac5f23cc356d6> with hop count 4
[2026-04-13 18:10:27] [Debug]    Rebroadcasting announce for <d62f88565958b2cd1045cf7f74796ce8> with hop count 4
[2026-04-13 18:10:27] [Debug]    Rebroadcasting announce for <020a777c04438f287caa9df8c6e99834> with hop count 4
[2026-04-13 18:10:27] [Debug]    Rebroadcasting announce for <985c7a6f4ed73219006b92b30b8b282c> with hop count 4
[2026-04-13 18:10:27] [Debug]    Rebroadcasting announce for <192eed7af8e3311445372f2a43cb63ec> with hop count 2
[2026-04-13 18:10:27] [Debug]    Rebroadcasting announce for <9c4780953ea0297e80cc1797ffe36deb> with hop count 4
[2026-04-13 18:10:27] [Debug]    Rebroadcasting announce for <8cdee9fb325052d66800632d7d3773b0> with hop count 4
[2026-04-13 18:10:27] [Debug]    Rebroadcasting announce for <44213278b52d888683e970004dc95f3c> with hop count 4
[2026-04-13 18:10:27] [Debug]    Rebroadcasting announce for <b993f1def2a655d9ef7b723f14a4d7b4> with hop count 4
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.81ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.16ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.57ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.55ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.52ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.51ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.48ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.47ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.46ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.43ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.42ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.4ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.38ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.37ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.35ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.33ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.32ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.3ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.28ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.26ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.25ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 8) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.23ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 8) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.21ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 8) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.2ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 9) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.12ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 9) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.1ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 9) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.08ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 10) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.05ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 10) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.04ms
[2026-04-13 18:10:27] [Extra]    Added announce to queue (height 10) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.03ms
[2026-04-13 18:10:28] [Extra]    Valid announce for <1f163087a2d335a87bd8dd4ceaf162f2> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:28] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:28] [Extra]    Heard a rebroadcast of announce for <02aaf088472435718061211d3752c8ed> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:28] [Extra]    Valid announce for <d20a381e1b46d0855c46105b565ec8ce> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:28] [Extra]    Heard a rebroadcast of announce for <d20a381e1b46d0855c46105b565ec8ce> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:28] [Extra]    Valid announce for <e59c163a01cac4628dace75d8a8c5efe> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:28] [Extra]    Heard a rebroadcast of announce for <e59c163a01cac4628dace75d8a8c5efe> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:28] [Extra]    Valid announce for <9f3a20b50352a2523193caa1b5130f5d> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:28] [Extra]    Valid announce for <219a60c23a74cf1ede2ee1c56dc790d7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:28] [Extra]    Heard a rebroadcast of announce for <219a60c23a74cf1ede2ee1c56dc790d7> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:28] [Extra]    Completed announce processing for <219a60c23a74cf1ede2ee1c56dc790d7>, local rebroadcast limit reached
[2026-04-13 18:10:28] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:28] [Extra]    Valid announce for <04535e4477fb3c573c6fa5abea7cc7d8> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:28] [Extra]    Remembering ratchet <9847480adb2f5f271e95> for <04535e4477fb3c573c6fa5abea7cc7d8>
[2026-04-13 18:10:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:10:28] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:28] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:10:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:28] [Debug]    Destination <04535e4477fb3c573c6fa5abea7cc7d8> is now 5 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:28] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-13 18:10:28] [Warning]  Attempting to reconnect...
[2026-04-13 18:10:28] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 18:10:28] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:28] [Extra]    Valid announce for <b993f1def2a655d9ef7b723f14a4d7b4> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:28] [Extra]    Valid announce for <13da3e301bfb15dc0d0499859e0e7cfe> 2 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:28] [Extra]    Valid announce for <ba2780f844f711525924923e9bfb23cb> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:28] [Extra]    Valid announce for <c62644119a8eb95abbe4647ead099d03> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:28] [Extra]    Valid announce for <6ee8d89ae74833c397169c07b81e62e2> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:28] [Debug]    Destination <6ee8d89ae74833c397169c07b81e62e2> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:28] [Extra]    Completed announce processing for <4ba30ddbc5e3639be796a98117bb08a8>, local rebroadcast limit reached
[2026-04-13 18:10:28] [Extra]    Completed announce processing for <e345f6220682e127cab52c3387436778>, local rebroadcast limit reached
[2026-04-13 18:10:28] [Extra]    Completed announce processing for <053db2ffc4105a601f6ac5f23cc356d6>, local rebroadcast limit reached
[2026-04-13 18:10:28] [Extra]    Completed announce processing for <d62f88565958b2cd1045cf7f74796ce8>, local rebroadcast limit reached
[2026-04-13 18:10:28] [Debug]    Rebroadcasting announce for <ca273d664d1a6c59a5a002670a641eff> with hop count 3
[2026-04-13 18:10:28] [Debug]    Rebroadcasting announce for <bb3819350c3c8ddaea5c441868e1699c> with hop count 4
[2026-04-13 18:10:28] [Debug]    Rebroadcasting announce for <d20a381e1b46d0855c46105b565ec8ce> with hop count 4
[2026-04-13 18:10:28] [Debug]    Rebroadcasting announce for <e59c163a01cac4628dace75d8a8c5efe> with hop count 4
[2026-04-13 18:10:28] [Debug]    Rebroadcasting announce for <02aaf088472435718061211d3752c8ed> with hop count 5
[2026-04-13 18:10:28] [Debug]    Rebroadcasting announce for <ba2780f844f711525924923e9bfb23cb> with hop count 5
[2026-04-13 18:10:28] [Debug]    Rebroadcasting announce for <c62644119a8eb95abbe4647ead099d03> with hop count 5
[2026-04-13 18:10:28] [Debug]    Rebroadcasting announce for <04535e4477fb3c573c6fa5abea7cc7d8> with hop count 5
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 11.58ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.13ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.75ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.56ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.52ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.51ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.48ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.46ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.45ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.42ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.41ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.39ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.37ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.35ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.34ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.32ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.26ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.22ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.18ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.17ms
[2026-04-13 18:10:28] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.16ms
[2026-04-13 18:10:29] [Extra]    Valid announce for <178b7b83ff68e7b8364ec6112ed4feb6> 6 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:29] [Extra]    Remembering ratchet <b2d1fce33ccfd63fbd83> for <178b7b83ff68e7b8364ec6112ed4feb6>
[2026-04-13 18:10:29] [Debug]    Destination <178b7b83ff68e7b8364ec6112ed4feb6> is now 6 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:29] [Extra]    Valid announce for <6ee8d89ae74833c397169c07b81e62e2> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:29] [Extra]    Heard a rebroadcast of announce for <6ee8d89ae74833c397169c07b81e62e2> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:29] [Debug]    Path request for <aa018ed28fb64405f8477866b78a668c> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:29] [Debug]    Ignoring path request for <aa018ed28fb64405f8477866b78a668c> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 18:10:29] [Extra]    Valid announce for <6ee8d89ae74833c397169c07b81e62e2> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:29] [Extra]    Heard a rebroadcast of announce for <6ee8d89ae74833c397169c07b81e62e2> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:29] [Debug]    Path request for <d7fa9b80db0ecbffe303616bebe78f74> on LocalInterface[59307]
[2026-04-13 18:10:29] [Debug]    Answering path request for <d7fa9b80db0ecbffe303616bebe78f74> on LocalInterface[59307], destination is local to this system
[2026-04-13 18:10:29] [Extra]    Completed announce processing for <ca273d664d1a6c59a5a002670a641eff>, local rebroadcast limit reached
[2026-04-13 18:10:29] [Extra]    Completed announce processing for <bb3819350c3c8ddaea5c441868e1699c>, local rebroadcast limit reached
[2026-04-13 18:10:29] [Extra]    Completed announce processing for <d20a381e1b46d0855c46105b565ec8ce>, local rebroadcast limit reached
[2026-04-13 18:10:29] [Extra]    Completed announce processing for <e59c163a01cac4628dace75d8a8c5efe>, local rebroadcast limit reached
[2026-04-13 18:10:29] [Extra]    Completed announce processing for <02aaf088472435718061211d3752c8ed>, local rebroadcast limit reached
[2026-04-13 18:10:29] [Debug]    Rebroadcasting announce for <1f163087a2d335a87bd8dd4ceaf162f2> with hop count 6
[2026-04-13 18:10:29] [Debug]    Rebroadcasting announce for <73400f494c8d580bd774443a5163127b> with hop count 3
[2026-04-13 18:10:29] [Debug]    Rebroadcasting announce for <9f3a20b50352a2523193caa1b5130f5d> with hop count 5
[2026-04-13 18:10:29] [Debug]    Rebroadcasting announce for <95cb09df57c070a2e9c6a35831e9904c> with hop count 10
[2026-04-13 18:10:29] [Debug]    Rebroadcasting announce for <6ee8d89ae74833c397169c07b81e62e2> with hop count 2
[2026-04-13 18:10:29] [Debug]    Rebroadcasting announce for <178b7b83ff68e7b8364ec6112ed4feb6> with hop count 6
[2026-04-13 18:10:29] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 13.96ms
[2026-04-13 18:10:29] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 9.5ms
[2026-04-13 18:10:29] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 1.11ms
[2026-04-13 18:10:29] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 0.8ms
[2026-04-13 18:10:29] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 0.77ms
[2026-04-13 18:10:29] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 0.76ms
[2026-04-13 18:10:29] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 0.71ms
[2026-04-13 18:10:29] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 0.69ms
[2026-04-13 18:10:29] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 0.68ms
[2026-04-13 18:10:29] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 0.62ms
[2026-04-13 18:10:29] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 0.6ms
[2026-04-13 18:10:29] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 0.59ms
[2026-04-13 18:10:29] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 0.39ms
[2026-04-13 18:10:29] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 0.38ms
[2026-04-13 18:10:29] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 0.36ms
[2026-04-13 18:10:30] [Extra]    Valid announce for <13da3e301bfb15dc0d0499859e0e7cfe> 2 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:30] [Debug]    Destination <13da3e301bfb15dc0d0499859e0e7cfe> is now 2 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:30] [Extra]    Completed announce processing for <1f163087a2d335a87bd8dd4ceaf162f2>, local rebroadcast limit reached
[2026-04-13 18:10:30] [Extra]    Completed announce processing for <73400f494c8d580bd774443a5163127b>, local rebroadcast limit reached
[2026-04-13 18:10:30] [Extra]    Completed announce processing for <9f3a20b50352a2523193caa1b5130f5d>, local rebroadcast limit reached
[2026-04-13 18:10:30] [Extra]    Completed announce processing for <95cb09df57c070a2e9c6a35831e9904c>, local rebroadcast limit reached
[2026-04-13 18:10:30] [Debug]    Rebroadcasting announce for <c077d6b4bf557e71faebf52aa54a74d2> with hop count 9
[2026-04-13 18:10:30] [Extra]    Valid announce for <13da3e301bfb15dc0d0499859e0e7cfe> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:31] [Extra]    Valid announce for <13da3e301bfb15dc0d0499859e0e7cfe> 2 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:31] [Extra]    Valid announce for <9a6c93b8d5c45f9c63a49625ba8cc134> 34 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:31] [Debug]    Destination <9a6c93b8d5c45f9c63a49625ba8cc134> is now 34 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:31] [Extra]    Valid announce for <fe495113c2090bdaf932b0f936b03864> 12 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:31] [Extra]    Remembering ratchet <6ab3fef84cf57abfb6f0> for <fe495113c2090bdaf932b0f936b03864>
[2026-04-13 18:10:31] [Debug]    Destination <fe495113c2090bdaf932b0f936b03864> is now 12 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:31] [Extra]    Completed announce processing for <c077d6b4bf557e71faebf52aa54a74d2>, local rebroadcast limit reached
[2026-04-13 18:10:31] [Debug]    Rebroadcasting announce for <794884194914d03c4e199d9c1f090b0c> with hop count 6
[2026-04-13 18:10:31] [Debug]    Rebroadcasting announce for <13da3e301bfb15dc0d0499859e0e7cfe> with hop count 2
[2026-04-13 18:10:31] [Debug]    Rebroadcasting announce for <9a6c93b8d5c45f9c63a49625ba8cc134> with hop count 34
[2026-04-13 18:10:31] [Debug]    Rebroadcasting announce for <fe495113c2090bdaf932b0f936b03864> with hop count 12
[2026-04-13 18:10:31] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.66ms
[2026-04-13 18:10:31] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.5ms
[2026-04-13 18:10:31] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.36ms
[2026-04-13 18:10:31] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.23ms
[2026-04-13 18:10:31] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.21ms
[2026-04-13 18:10:31] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.19ms
[2026-04-13 18:10:31] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.17ms
[2026-04-13 18:10:31] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.15ms
[2026-04-13 18:10:31] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.14ms
[2026-04-13 18:10:31] [Extra]    Valid announce for <fe495113c2090bdaf932b0f936b03864> 12 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:32] [Extra]    Valid announce for <f3a6a73294416a6d9e75706bb9167c6c> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:32] [Debug]    Destination <f3a6a73294416a6d9e75706bb9167c6c> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:32] [Extra]    Valid announce for <5789bdb62e037b74610b0a29fd586687> 10 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:32] [Debug]    Destination <5789bdb62e037b74610b0a29fd586687> is now 10 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:32] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:32] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:32] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:32] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:32] [Extra]    Completed announce processing for <794884194914d03c4e199d9c1f090b0c>, local rebroadcast limit reached
[2026-04-13 18:10:32] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 6
[2026-04-13 18:10:32] [Debug]    Rebroadcasting announce for <f3a6a73294416a6d9e75706bb9167c6c> with hop count 3
[2026-04-13 18:10:32] [Debug]    Rebroadcasting announce for <5789bdb62e037b74610b0a29fd586687> with hop count 10
[2026-04-13 18:10:32] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.22ms
[2026-04-13 18:10:32] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.12ms
[2026-04-13 18:10:32] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.07ms
[2026-04-13 18:10:32] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.02ms
[2026-04-13 18:10:32] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.01ms
[2026-04-13 18:10:32] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.01ms
[2026-04-13 18:10:32] [Extra]    Valid announce for <f3a6a73294416a6d9e75706bb9167c6c> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:32] [Extra]    Heard a rebroadcast of announce for <f3a6a73294416a6d9e75706bb9167c6c> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:32] [Extra]    Valid announce for <f3a6a73294416a6d9e75706bb9167c6c> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:32] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:32] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:10:33] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:33] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:10:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 18:10:33] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:33] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-13 18:10:33] [Warning]  Attempting to reconnect...
[2026-04-13 18:10:33] [Extra]    Completed announce processing for <2d8a25919ea488ce008d3635d9b104c7>, local rebroadcast limit reached
[2026-04-13 18:10:33] [Debug]    Rebroadcasting announce for <020a777c04438f287caa9df8c6e99834> with hop count 4
[2026-04-13 18:10:33] [Debug]    Rebroadcasting announce for <985c7a6f4ed73219006b92b30b8b282c> with hop count 4
[2026-04-13 18:10:33] [Debug]    Rebroadcasting announce for <192eed7af8e3311445372f2a43cb63ec> with hop count 2
[2026-04-13 18:10:33] [Debug]    Rebroadcasting announce for <9c4780953ea0297e80cc1797ffe36deb> with hop count 4
[2026-04-13 18:10:33] [Debug]    Rebroadcasting announce for <8cdee9fb325052d66800632d7d3773b0> with hop count 4
[2026-04-13 18:10:33] [Debug]    Rebroadcasting announce for <44213278b52d888683e970004dc95f3c> with hop count 4
[2026-04-13 18:10:33] [Debug]    Rebroadcasting announce for <b993f1def2a655d9ef7b723f14a4d7b4> with hop count 4
[2026-04-13 18:10:33] [Debug]    Rebroadcasting announce for <ca273d664d1a6c59a5a002670a641eff> with hop count 3
[2026-04-13 18:10:33] [Debug]    Rebroadcasting announce for <73400f494c8d580bd774443a5163127b> with hop count 3
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.73ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.51ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.4ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.23ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.21ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.2ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.17ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.15ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.14ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.12ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.1ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.09ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.07ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.06ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.04ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.02ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.01ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.99ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.97ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.96ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.93ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 8) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.9ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 8) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.89ms
[2026-04-13 18:10:33] [Extra]    Added announce to queue (height 8) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.88ms
[2026-04-13 18:10:34] [Extra]    Valid announce for <8de2bfef94c263a7de4b026fdfed72c9> 65 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:34] [Debug]    Destination <8de2bfef94c263a7de4b026fdfed72c9> is now 65 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:34] [Extra]    Valid announce for <2c6ccd83044bf6397fe4bacc3288f7ea> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:34] [Debug]    Destination <2c6ccd83044bf6397fe4bacc3288f7ea> is now 5 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:34] [Extra]    Valid announce for <2c6ccd83044bf6397fe4bacc3288f7ea> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:34] [Extra]    Heard a rebroadcast of announce for <2c6ccd83044bf6397fe4bacc3288f7ea> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:34] [Extra]    Completed announce processing for <020a777c04438f287caa9df8c6e99834>, local rebroadcast limit reached
[2026-04-13 18:10:34] [Extra]    Completed announce processing for <985c7a6f4ed73219006b92b30b8b282c>, local rebroadcast limit reached
[2026-04-13 18:10:34] [Extra]    Completed announce processing for <192eed7af8e3311445372f2a43cb63ec>, local rebroadcast limit reached
[2026-04-13 18:10:34] [Extra]    Completed announce processing for <9c4780953ea0297e80cc1797ffe36deb>, local rebroadcast limit reached
[2026-04-13 18:10:34] [Extra]    Completed announce processing for <8cdee9fb325052d66800632d7d3773b0>, local rebroadcast limit reached
[2026-04-13 18:10:34] [Extra]    Completed announce processing for <44213278b52d888683e970004dc95f3c>, local rebroadcast limit reached
[2026-04-13 18:10:34] [Extra]    Completed announce processing for <b993f1def2a655d9ef7b723f14a4d7b4>, local rebroadcast limit reached
[2026-04-13 18:10:34] [Debug]    Rebroadcasting announce for <ba2780f844f711525924923e9bfb23cb> with hop count 5
[2026-04-13 18:10:34] [Debug]    Rebroadcasting announce for <c62644119a8eb95abbe4647ead099d03> with hop count 5
[2026-04-13 18:10:34] [Debug]    Rebroadcasting announce for <04535e4477fb3c573c6fa5abea7cc7d8> with hop count 5
[2026-04-13 18:10:34] [Debug]    Rebroadcasting announce for <8de2bfef94c263a7de4b026fdfed72c9> with hop count 65
[2026-04-13 18:10:34] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.72ms
[2026-04-13 18:10:34] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 2.83ms
[2026-04-13 18:10:34] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 1.91ms
[2026-04-13 18:10:34] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 1.49ms
[2026-04-13 18:10:34] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 1.46ms
[2026-04-13 18:10:34] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 1.45ms
[2026-04-13 18:10:34] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 1.42ms
[2026-04-13 18:10:34] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 1.41ms
[2026-04-13 18:10:34] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 1.39ms
[2026-04-13 18:10:35] [Extra]    Valid announce for <c24169e0e90677f5a1839577742e5d23> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:35] [Debug]    Destination <c24169e0e90677f5a1839577742e5d23> is now 5 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:35] [Extra]    Valid announce for <8de2bfef94c263a7de4b026fdfed72c9> 67 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:35] [Extra]    Rebroadcasted announce for <8de2bfef94c263a7de4b026fdfed72c9> has been passed on to another node, no further tries needed
[2026-04-13 18:10:35] [Extra]    Valid announce for <c24169e0e90677f5a1839577742e5d23> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:35] [Extra]    Completed announce processing for <ba2780f844f711525924923e9bfb23cb>, local rebroadcast limit reached
[2026-04-13 18:10:35] [Extra]    Completed announce processing for <c62644119a8eb95abbe4647ead099d03>, local rebroadcast limit reached
[2026-04-13 18:10:35] [Extra]    Completed announce processing for <04535e4477fb3c573c6fa5abea7cc7d8>, local rebroadcast limit reached
[2026-04-13 18:10:35] [Debug]    Rebroadcasting announce for <6ee8d89ae74833c397169c07b81e62e2> with hop count 2
[2026-04-13 18:10:35] [Debug]    Rebroadcasting announce for <178b7b83ff68e7b8364ec6112ed4feb6> with hop count 6
[2026-04-13 18:10:35] [Debug]    Rebroadcasting announce for <2c6ccd83044bf6397fe4bacc3288f7ea> with hop count 5
[2026-04-13 18:10:35] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.21ms
[2026-04-13 18:10:35] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.13ms
[2026-04-13 18:10:35] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.08ms
[2026-04-13 18:10:35] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.02ms
[2026-04-13 18:10:35] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.02ms
[2026-04-13 18:10:35] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.01ms
[2026-04-13 18:10:36] [Extra]    Valid announce for <c24169e0e90677f5a1839577742e5d23> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:36] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:36] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:36] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:36] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:36] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:36] [Extra]    Completed announce processing for <6ee8d89ae74833c397169c07b81e62e2>, local rebroadcast limit reached
[2026-04-13 18:10:36] [Extra]    Completed announce processing for <178b7b83ff68e7b8364ec6112ed4feb6>, local rebroadcast limit reached
[2026-04-13 18:10:36] [Debug]    Rebroadcasting announce for <c24169e0e90677f5a1839577742e5d23> with hop count 5
[2026-04-13 18:10:36] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 4
[2026-04-13 18:10:36] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.16ms
[2026-04-13 18:10:36] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.06ms
[2026-04-13 18:10:36] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.99ms
[2026-04-13 18:10:36] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:36] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:37] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:37] [Extra]    Valid announce for <a64477187d9dcec0f9317b97096039f7> 8 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:37] [Extra]    Remembering ratchet <a6453384682661a87f9a> for <a64477187d9dcec0f9317b97096039f7>
[2026-04-13 18:10:37] [Debug]    Destination <a64477187d9dcec0f9317b97096039f7> is now 8 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:37] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:37] [Extra]    Valid announce for <4ba30ddbc5e3639be796a98117bb08a8> 8 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:37] [Debug]    Rebroadcasting announce for <13da3e301bfb15dc0d0499859e0e7cfe> with hop count 2
[2026-04-13 18:10:37] [Debug]    Rebroadcasting announce for <9a6c93b8d5c45f9c63a49625ba8cc134> with hop count 34
[2026-04-13 18:10:37] [Debug]    Rebroadcasting announce for <fe495113c2090bdaf932b0f936b03864> with hop count 12
[2026-04-13 18:10:37] [Debug]    Rebroadcasting announce for <ca273d664d1a6c59a5a002670a641eff> with hop count 3
[2026-04-13 18:10:37] [Debug]    Rebroadcasting announce for <73400f494c8d580bd774443a5163127b> with hop count 3
[2026-04-13 18:10:37] [Debug]    Rebroadcasting announce for <a64477187d9dcec0f9317b97096039f7> with hop count 8
[2026-04-13 18:10:37] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.02ms
[2026-04-13 18:10:37] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.76ms
[2026-04-13 18:10:37] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.65ms
[2026-04-13 18:10:37] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.53ms
[2026-04-13 18:10:37] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.51ms
[2026-04-13 18:10:37] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.5ms
[2026-04-13 18:10:37] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.47ms
[2026-04-13 18:10:37] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.46ms
[2026-04-13 18:10:37] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.44ms
[2026-04-13 18:10:37] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.42ms
[2026-04-13 18:10:37] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.3ms
[2026-04-13 18:10:37] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.29ms
[2026-04-13 18:10:37] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.26ms
[2026-04-13 18:10:37] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.24ms
[2026-04-13 18:10:37] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.23ms
[2026-04-13 18:10:38] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:38] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:10:38] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:38] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:10:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:38] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-13 18:10:38] [Warning]  Attempting to reconnect...
[2026-04-13 18:10:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 18:10:38] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:38] [Extra]    Valid announce for <636d51787cd3b5f8e90aec12fb6e6b7a> 8 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:38] [Debug]    Destination <636d51787cd3b5f8e90aec12fb6e6b7a> is now 8 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:38] [Extra]    Valid announce for <a64477187d9dcec0f9317b97096039f7> 9 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:38] [Extra]    Heard a rebroadcast of announce for <a64477187d9dcec0f9317b97096039f7> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:38] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:38] [Extra]    Valid announce for <f3a6a73294416a6d9e75706bb9167c6c> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:38] [Extra]    Valid announce for <d62f88565958b2cd1045cf7f74796ce8> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:38] [Extra]    Valid announce for <636d51787cd3b5f8e90aec12fb6e6b7a> 8 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:38] [Extra]    Completed announce processing for <13da3e301bfb15dc0d0499859e0e7cfe>, local rebroadcast limit reached
[2026-04-13 18:10:38] [Extra]    Completed announce processing for <9a6c93b8d5c45f9c63a49625ba8cc134>, local rebroadcast limit reached
[2026-04-13 18:10:38] [Extra]    Completed announce processing for <fe495113c2090bdaf932b0f936b03864>, local rebroadcast limit reached
[2026-04-13 18:10:38] [Debug]    Rebroadcasting announce for <f3a6a73294416a6d9e75706bb9167c6c> with hop count 3
[2026-04-13 18:10:38] [Debug]    Rebroadcasting announce for <5789bdb62e037b74610b0a29fd586687> with hop count 10
[2026-04-13 18:10:38] [Debug]    Rebroadcasting announce for <110d7f3159c1d306851c3ec5c6d302ef> with hop count 4
[2026-04-13 18:10:38] [Debug]    Rebroadcasting announce for <636d51787cd3b5f8e90aec12fb6e6b7a> with hop count 8
[2026-04-13 18:10:38] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.14ms
[2026-04-13 18:10:38] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.69ms
[2026-04-13 18:10:38] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.53ms
[2026-04-13 18:10:38] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.38ms
[2026-04-13 18:10:38] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.35ms
[2026-04-13 18:10:38] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.33ms
[2026-04-13 18:10:38] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.29ms
[2026-04-13 18:10:38] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.27ms
[2026-04-13 18:10:38] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.25ms
[2026-04-13 18:10:39] [Extra]    Valid announce for <2d84b2297ec68336c31d45e34be62f8c> 6 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:39] [Extra]    Remembering ratchet <3c883a5cc91c98c2b8b8> for <2d84b2297ec68336c31d45e34be62f8c>
[2026-04-13 18:10:39] [Debug]    Destination <2d84b2297ec68336c31d45e34be62f8c> is now 6 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:39] [Extra]    Completed announce processing for <f3a6a73294416a6d9e75706bb9167c6c>, local rebroadcast limit reached
[2026-04-13 18:10:39] [Extra]    Completed announce processing for <5789bdb62e037b74610b0a29fd586687>, local rebroadcast limit reached
[2026-04-13 18:10:39] [Debug]    Rebroadcasting announce for <2d84b2297ec68336c31d45e34be62f8c> with hop count 6
[2026-04-13 18:10:40] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:40] [Debug]    Destination <e80bd281bd26e00d735aada7b7b94c7a> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:40] [Extra]    Valid announce for <db99465d104f1f2b72739c9db4347797> 8 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:40] [Debug]    Destination <db99465d104f1f2b72739c9db4347797> is now 8 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:40] [Extra]    Valid announce for <16cf0ecc7fd12831b31cddc5a909c5ec> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:40] [Extra]    Remembering ratchet <3cfc1ec5f84280f15c40> for <16cf0ecc7fd12831b31cddc5a909c5ec>
[2026-04-13 18:10:40] [Debug]    Destination <16cf0ecc7fd12831b31cddc5a909c5ec> is now 5 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:40] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:40] [Debug]    Rebroadcasting announce for <e80bd281bd26e00d735aada7b7b94c7a> with hop count 4
[2026-04-13 18:10:40] [Debug]    Rebroadcasting announce for <16cf0ecc7fd12831b31cddc5a909c5ec> with hop count 5
[2026-04-13 18:10:40] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.75ms
[2026-04-13 18:10:40] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.57ms
[2026-04-13 18:10:40] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.46ms
[2026-04-13 18:10:41] [Extra]    Valid announce for <16cf0ecc7fd12831b31cddc5a909c5ec> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:41] [Extra]    Valid announce for <16cf0ecc7fd12831b31cddc5a909c5ec> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:41] [Extra]    Heard a rebroadcast of announce for <16cf0ecc7fd12831b31cddc5a909c5ec> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:41] [Debug]    Rebroadcasting announce for <2c6ccd83044bf6397fe4bacc3288f7ea> with hop count 5
[2026-04-13 18:10:41] [Debug]    Rebroadcasting announce for <db99465d104f1f2b72739c9db4347797> with hop count 8
[2026-04-13 18:10:41] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 12.61ms
[2026-04-13 18:10:41] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 12.17ms
[2026-04-13 18:10:41] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 11.9ms
[2026-04-13 18:10:42] [Extra]    Valid announce for <75a761de7bdd03adeb6ad16b004e53a1> 6 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:42] [Debug]    Destination <75a761de7bdd03adeb6ad16b004e53a1> is now 6 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:42] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:42] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:42] [Extra]    Valid announce for <75a761de7bdd03adeb6ad16b004e53a1> 7 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:42] [Extra]    Heard a rebroadcast of announce for <75a761de7bdd03adeb6ad16b004e53a1> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:42] [Extra]    Completed announce processing for <2c6ccd83044bf6397fe4bacc3288f7ea>, local rebroadcast limit reached
[2026-04-13 18:10:42] [Debug]    Rebroadcasting announce for <c24169e0e90677f5a1839577742e5d23> with hop count 5
[2026-04-13 18:10:42] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 4
[2026-04-13 18:10:42] [Debug]    Rebroadcasting announce for <110d7f3159c1d306851c3ec5c6d302ef> with hop count 4
[2026-04-13 18:10:42] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.92ms
[2026-04-13 18:10:42] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.54ms
[2026-04-13 18:10:42] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.29ms
[2026-04-13 18:10:42] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.06ms
[2026-04-13 18:10:42] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.01ms
[2026-04-13 18:10:42] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.93ms
[2026-04-13 18:10:43] [Extra]    Valid announce for <75a761de7bdd03adeb6ad16b004e53a1> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:43] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:10:43] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:43] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:10:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 18:10:43] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:43] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-13 18:10:43] [Warning]  Attempting to reconnect...
[2026-04-13 18:10:43] [Debug]    Rebroadcasting announce for <ca273d664d1a6c59a5a002670a641eff> with hop count 3
[2026-04-13 18:10:43] [Debug]    Rebroadcasting announce for <73400f494c8d580bd774443a5163127b> with hop count 3
[2026-04-13 18:10:43] [Extra]    Completed announce processing for <c24169e0e90677f5a1839577742e5d23>, local rebroadcast limit reached
[2026-04-13 18:10:43] [Extra]    Completed announce processing for <2d8a25919ea488ce008d3635d9b104c7>, local rebroadcast limit reached
[2026-04-13 18:10:43] [Debug]    Rebroadcasting announce for <a64477187d9dcec0f9317b97096039f7> with hop count 8
[2026-04-13 18:10:43] [Debug]    Rebroadcasting announce for <75a761de7bdd03adeb6ad16b004e53a1> with hop count 6
[2026-04-13 18:10:43] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 11.23ms
[2026-04-13 18:10:43] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 11.02ms
[2026-04-13 18:10:43] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 10.73ms
[2026-04-13 18:10:43] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 10.6ms
[2026-04-13 18:10:43] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 10.56ms
[2026-04-13 18:10:43] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 10.49ms
[2026-04-13 18:10:43] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 10.45ms
[2026-04-13 18:10:43] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 10.43ms
[2026-04-13 18:10:43] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 10.42ms
[2026-04-13 18:10:44] [Extra]    Valid announce for <94bd1825f5818f4839b276eefbfc99a9> 7 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:44] [Debug]    Destination <94bd1825f5818f4839b276eefbfc99a9> is now 7 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:45] [Extra]    Completed announce processing for <ca273d664d1a6c59a5a002670a641eff>, local rebroadcast limit reached
[2026-04-13 18:10:45] [Extra]    Completed announce processing for <73400f494c8d580bd774443a5163127b>, local rebroadcast limit reached
[2026-04-13 18:10:45] [Extra]    Completed announce processing for <a64477187d9dcec0f9317b97096039f7>, local rebroadcast limit reached
[2026-04-13 18:10:45] [Debug]    Rebroadcasting announce for <636d51787cd3b5f8e90aec12fb6e6b7a> with hop count 8
[2026-04-13 18:10:45] [Debug]    Rebroadcasting announce for <94bd1825f5818f4839b276eefbfc99a9> with hop count 7
[2026-04-13 18:10:45] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.56ms
[2026-04-13 18:10:45] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.23ms
[2026-04-13 18:10:45] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.0ms
[2026-04-13 18:10:45] [Extra]    Valid announce for <477fddb189b7a04d81c5964809a5980b> 13 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:45] [Extra]    Remembering ratchet <4f6fa8a8191b421342cc> for <477fddb189b7a04d81c5964809a5980b>
[2026-04-13 18:10:45] [Debug]    Destination <477fddb189b7a04d81c5964809a5980b> is now 13 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:46] [Extra]    Completed announce processing for <636d51787cd3b5f8e90aec12fb6e6b7a>, local rebroadcast limit reached
[2026-04-13 18:10:46] [Debug]    Rebroadcasting announce for <2d84b2297ec68336c31d45e34be62f8c> with hop count 6
[2026-04-13 18:10:46] [Debug]    Rebroadcasting announce for <477fddb189b7a04d81c5964809a5980b> with hop count 13
[2026-04-13 18:10:46] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 9.06ms
[2026-04-13 18:10:46] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.8ms
[2026-04-13 18:10:46] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.68ms
[2026-04-13 18:10:46] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:46] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:47] [Extra]    Completed announce processing for <2d84b2297ec68336c31d45e34be62f8c>, local rebroadcast limit reached
[2026-04-13 18:10:47] [Debug]    Rebroadcasting announce for <e80bd281bd26e00d735aada7b7b94c7a> with hop count 4
[2026-04-13 18:10:47] [Debug]    Rebroadcasting announce for <16cf0ecc7fd12831b31cddc5a909c5ec> with hop count 5
[2026-04-13 18:10:47] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 4
[2026-04-13 18:10:47] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.34ms
[2026-04-13 18:10:47] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.0ms
[2026-04-13 18:10:47] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.89ms
[2026-04-13 18:10:47] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.74ms
[2026-04-13 18:10:47] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.72ms
[2026-04-13 18:10:47] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.71ms
[2026-04-13 18:10:47] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:48] [Extra]    Completed announce processing for <e80bd281bd26e00d735aada7b7b94c7a>, local rebroadcast limit reached
[2026-04-13 18:10:48] [Debug]    Rebroadcasting announce for <db99465d104f1f2b72739c9db4347797> with hop count 8
[2026-04-13 18:10:48] [Extra]    Completed announce processing for <16cf0ecc7fd12831b31cddc5a909c5ec>, local rebroadcast limit reached
[2026-04-13 18:10:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:10:48] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:48] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:10:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:48] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-13 18:10:48] [Warning]  Attempting to reconnect...
[2026-04-13 18:10:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 18:10:48] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:48] [Debug]    Path request for <a3dfea289534e4b2d0f5730310eebd99> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:48] [Debug]    Ignoring path request for <a3dfea289534e4b2d0f5730310eebd99> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 18:10:48] [Extra]    Valid announce for <424157ea5e1bb6dd1aa6dba8970460cf> 38 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:48] [Extra]    Remembering ratchet <1e84fe0c6684ccc9ce8c> for <424157ea5e1bb6dd1aa6dba8970460cf>
[2026-04-13 18:10:48] [Debug]    Destination <424157ea5e1bb6dd1aa6dba8970460cf> is now 38 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:49] [Debug]    Rebroadcasting announce for <110d7f3159c1d306851c3ec5c6d302ef> with hop count 4
[2026-04-13 18:10:49] [Extra]    Completed announce processing for <db99465d104f1f2b72739c9db4347797>, local rebroadcast limit reached
[2026-04-13 18:10:49] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:49] [Debug]    Destination <29b2ebe588859e48aabf13e97cfe245b> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:49] [Extra]    Valid announce for <1ba3f953b8e28bd2f5a5ec2e741edf65> 8 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:50] [Extra]    Completed announce processing for <110d7f3159c1d306851c3ec5c6d302ef>, local rebroadcast limit reached
[2026-04-13 18:10:50] [Debug]    Rebroadcasting announce for <75a761de7bdd03adeb6ad16b004e53a1> with hop count 6
[2026-04-13 18:10:50] [Debug]    Rebroadcasting announce for <424157ea5e1bb6dd1aa6dba8970460cf> with hop count 38
[2026-04-13 18:10:50] [Debug]    Rebroadcasting announce for <29b2ebe588859e48aabf13e97cfe245b> with hop count 2
[2026-04-13 18:10:50] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.41ms
[2026-04-13 18:10:50] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.32ms
[2026-04-13 18:10:50] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.26ms
[2026-04-13 18:10:50] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.2ms
[2026-04-13 18:10:50] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.19ms
[2026-04-13 18:10:50] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.18ms
[2026-04-13 18:10:50] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:50] [Extra]    Heard a rebroadcast of announce for <29b2ebe588859e48aabf13e97cfe245b> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:50] [Extra]    Valid announce for <9eb0f17e1691e819bb34eddafa2ca82b> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:50] [Debug]    Destination <9eb0f17e1691e819bb34eddafa2ca82b> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:51] [Extra]    Completed announce processing for <75a761de7bdd03adeb6ad16b004e53a1>, local rebroadcast limit reached
[2026-04-13 18:10:51] [Debug]    Rebroadcasting announce for <94bd1825f5818f4839b276eefbfc99a9> with hop count 7
[2026-04-13 18:10:51] [Extra]    Valid announce for <9eb0f17e1691e819bb34eddafa2ca82b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:51] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:51] [Debug]    Destination <e345f6220682e127cab52c3387436778> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:51] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:51] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:51] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:51] [Debug]    Destination <02aaf088472435718061211d3752c8ed> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:51] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:51] [Extra]    Valid announce for <f202ae6541f5e69c204d0b2bcbfcd273> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:51] [Extra]    Remembering ratchet <5bb71ae7f949fbd38417> for <f202ae6541f5e69c204d0b2bcbfcd273>
[2026-04-13 18:10:51] [Debug]    Destination <f202ae6541f5e69c204d0b2bcbfcd273> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:51] [Extra]    Valid announce for <103eb3c7f35278ba33e7d014e341b3ec> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:51] [Debug]    Destination <103eb3c7f35278ba33e7d014e341b3ec> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:51] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:51] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:51] [Debug]    Path request for <3b171e0b79acf468ae1bf3a6d8515d12> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:51] [Debug]    Not answering path request for <3b171e0b79acf468ae1bf3a6d8515d12> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242], since next hop is the requestor
[2026-04-13 18:10:52] [Extra]    Completed announce processing for <94bd1825f5818f4839b276eefbfc99a9>, local rebroadcast limit reached
[2026-04-13 18:10:52] [Debug]    Rebroadcasting announce for <477fddb189b7a04d81c5964809a5980b> with hop count 13
[2026-04-13 18:10:52] [Debug]    Rebroadcasting announce for <9eb0f17e1691e819bb34eddafa2ca82b> with hop count 3
[2026-04-13 18:10:52] [Debug]    Rebroadcasting announce for <e345f6220682e127cab52c3387436778> with hop count 5
[2026-04-13 18:10:52] [Debug]    Rebroadcasting announce for <73400f494c8d580bd774443a5163127b> with hop count 3
[2026-04-13 18:10:52] [Debug]    Rebroadcasting announce for <02aaf088472435718061211d3752c8ed> with hop count 4
[2026-04-13 18:10:52] [Debug]    Rebroadcasting announce for <f202ae6541f5e69c204d0b2bcbfcd273> with hop count 4
[2026-04-13 18:10:52] [Debug]    Rebroadcasting announce for <103eb3c7f35278ba33e7d014e341b3ec> with hop count 4
[2026-04-13 18:10:52] [Debug]    Rebroadcasting announce for <ca273d664d1a6c59a5a002670a641eff> with hop count 3
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.57ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.25ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.1ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.92ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.9ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.88ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.84ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.82ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.81ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.77ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.76ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.74ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.7ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.68ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.67ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.63ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.62ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.6ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.55ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.53ms
[2026-04-13 18:10:52] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.52ms
[2026-04-13 18:10:52] [Extra]    Valid announce for <fb31704d247f8b1547f9a3b356760e7b> 10 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:52] [Debug]    Destination <fb31704d247f8b1547f9a3b356760e7b> is now 10 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:52] [Extra]    Valid announce for <dc085bf490f24f9051a492155bf5e283> 8 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:52] [Debug]    Destination <dc085bf490f24f9051a492155bf5e283> is now 8 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:52] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:52] [Extra]    Heard a rebroadcast of announce for <73400f494c8d580bd774443a5163127b> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:52] [Extra]    Valid announce for <54f0e7796ac804890832cb3ee61131f2> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:52] [Extra]    Remembering ratchet <38ebfa21cea03497c2d7> for <54f0e7796ac804890832cb3ee61131f2>
[2026-04-13 18:10:52] [Debug]    Destination <54f0e7796ac804890832cb3ee61131f2> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:52] [Extra]    Valid announce for <dce1135780d75f28804a92545aea418a> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:52] [Debug]    Destination <dce1135780d75f28804a92545aea418a> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:52] [Extra]    Valid announce for <f202ae6541f5e69c204d0b2bcbfcd273> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:52] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:52] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:52] [Extra]    Heard a rebroadcast of announce for <02aaf088472435718061211d3752c8ed> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:52] [Extra]    Valid announce for <103eb3c7f35278ba33e7d014e341b3ec> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:52] [Extra]    Heard a rebroadcast of announce for <103eb3c7f35278ba33e7d014e341b3ec> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:52] [Extra]    Valid announce for <dce1135780d75f28804a92545aea418a> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:52] [Extra]    Valid announce for <54f0e7796ac804890832cb3ee61131f2> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:52] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:52] [Extra]    Valid announce for <54f0e7796ac804890832cb3ee61131f2> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:52] [Extra]    Valid announce for <12cb1ed29943213839f0b0d18cd42761> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:52] [Debug]    Destination <12cb1ed29943213839f0b0d18cd42761> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:53] [Extra]    Completed announce processing for <477fddb189b7a04d81c5964809a5980b>, local rebroadcast limit reached
[2026-04-13 18:10:53] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 4
[2026-04-13 18:10:53] [Debug]    Rebroadcasting announce for <fb31704d247f8b1547f9a3b356760e7b> with hop count 10
[2026-04-13 18:10:53] [Debug]    Rebroadcasting announce for <dc085bf490f24f9051a492155bf5e283> with hop count 8
[2026-04-13 18:10:53] [Debug]    Rebroadcasting announce for <54f0e7796ac804890832cb3ee61131f2> with hop count 4
[2026-04-13 18:10:53] [Debug]    Rebroadcasting announce for <dce1135780d75f28804a92545aea418a> with hop count 4
[2026-04-13 18:10:53] [Debug]    Rebroadcasting announce for <12cb1ed29943213839f0b0d18cd42761> with hop count 4
[2026-04-13 18:10:53] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.76ms
[2026-04-13 18:10:53] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.3ms
[2026-04-13 18:10:53] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.99ms
[2026-04-13 18:10:53] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.74ms
[2026-04-13 18:10:53] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.69ms
[2026-04-13 18:10:53] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.66ms
[2026-04-13 18:10:53] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.61ms
[2026-04-13 18:10:53] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.58ms
[2026-04-13 18:10:53] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.56ms
[2026-04-13 18:10:53] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.51ms
[2026-04-13 18:10:53] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.49ms
[2026-04-13 18:10:53] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.46ms
[2026-04-13 18:10:53] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.42ms
[2026-04-13 18:10:53] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.39ms
[2026-04-13 18:10:53] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.37ms
[2026-04-13 18:10:53] [Extra]    Valid announce for <768fb269391fb64afc979dffe6572cb2> 13 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:53] [Debug]    Destination <768fb269391fb64afc979dffe6572cb2> is now 13 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:53] [Extra]    Valid announce for <12cb1ed29943213839f0b0d18cd42761> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:53] [Extra]    Valid announce for <768fb269391fb64afc979dffe6572cb2> 13 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:53] [Extra]    Valid announce for <587a35e51879b1954788cca5c3be2ff7> 11 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:53] [Debug]    Destination <587a35e51879b1954788cca5c3be2ff7> is now 11 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:53] [Extra]    Valid announce for <34593130ea32ad163dff5b3066c43964> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:53] [Debug]    Destination <34593130ea32ad163dff5b3066c43964> is now 5 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:10:53] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:53] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:10:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 18:10:53] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:53] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-13 18:10:53] [Warning]  Attempting to reconnect...
[2026-04-13 18:10:53] [Extra]    Valid announce for <4dd9ed5f1d35aa5e95f9e34477bef74d> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:53] [Debug]    Destination <4dd9ed5f1d35aa5e95f9e34477bef74d> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:53] [Extra]    Valid announce for <34593130ea32ad163dff5b3066c43964> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:54] [Extra]    Completed announce processing for <2d8a25919ea488ce008d3635d9b104c7>, local rebroadcast limit reached
[2026-04-13 18:10:54] [Debug]    Rebroadcasting announce for <768fb269391fb64afc979dffe6572cb2> with hop count 13
[2026-04-13 18:10:54] [Debug]    Rebroadcasting announce for <587a35e51879b1954788cca5c3be2ff7> with hop count 11
[2026-04-13 18:10:54] [Debug]    Rebroadcasting announce for <34593130ea32ad163dff5b3066c43964> with hop count 5
[2026-04-13 18:10:54] [Debug]    Rebroadcasting announce for <4dd9ed5f1d35aa5e95f9e34477bef74d> with hop count 3
[2026-04-13 18:10:54] [Extra]    Released 1 link
[2026-04-13 18:10:54] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.22ms
[2026-04-13 18:10:54] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.12ms
[2026-04-13 18:10:54] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.06ms
[2026-04-13 18:10:54] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.98ms
[2026-04-13 18:10:54] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.97ms
[2026-04-13 18:10:54] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.96ms
[2026-04-13 18:10:54] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.94ms
[2026-04-13 18:10:54] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.93ms
[2026-04-13 18:10:54] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.93ms
[2026-04-13 18:10:54] [Extra]    Valid announce for <4dd9ed5f1d35aa5e95f9e34477bef74d> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:54] [Extra]    Heard a rebroadcast of announce for <4dd9ed5f1d35aa5e95f9e34477bef74d> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:54] [Extra]    Valid announce for <3c81447dff85b425c79ca5a97ff75f75> 9 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:54] [Debug]    Destination <3c81447dff85b425c79ca5a97ff75f75> is now 9 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:54] [Extra]    Valid announce for <38073923c15b25893cd38a7938a9943a> 10 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:54] [Extra]    Remembering ratchet <90fecec03e5de4bb72d9> for <38073923c15b25893cd38a7938a9943a>
[2026-04-13 18:10:54] [Debug]    Destination <38073923c15b25893cd38a7938a9943a> is now 10 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:54] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:54] [Debug]    Destination <794884194914d03c4e199d9c1f090b0c> is now 6 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:55] [Debug]    Rebroadcasting announce for <3c81447dff85b425c79ca5a97ff75f75> with hop count 9
[2026-04-13 18:10:55] [Debug]    Rebroadcasting announce for <38073923c15b25893cd38a7938a9943a> with hop count 10
[2026-04-13 18:10:55] [Debug]    Rebroadcasting announce for <794884194914d03c4e199d9c1f090b0c> with hop count 6
[2026-04-13 18:10:55] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.16ms
[2026-04-13 18:10:55] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.81ms
[2026-04-13 18:10:55] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.69ms
[2026-04-13 18:10:55] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.58ms
[2026-04-13 18:10:55] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.56ms
[2026-04-13 18:10:55] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.55ms
[2026-04-13 18:10:55] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 7 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:55] [Extra]    Heard a rebroadcast of announce for <794884194914d03c4e199d9c1f090b0c> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:55] [Extra]    Valid announce for <38073923c15b25893cd38a7938a9943a> 11 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:55] [Extra]    Heard a rebroadcast of announce for <38073923c15b25893cd38a7938a9943a> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:55] [Extra]    Valid announce for <55c16056b1b0f93042c92ea31ceb801a> 16 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:55] [Extra]    Remembering ratchet <9c7f806698ce1cee4d34> for <55c16056b1b0f93042c92ea31ceb801a>
[2026-04-13 18:10:55] [Debug]    Destination <55c16056b1b0f93042c92ea31ceb801a> is now 16 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:55] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:55] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:55] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:55] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:56] [Debug]    Rebroadcasting announce for <424157ea5e1bb6dd1aa6dba8970460cf> with hop count 38
[2026-04-13 18:10:56] [Debug]    Rebroadcasting announce for <29b2ebe588859e48aabf13e97cfe245b> with hop count 2
[2026-04-13 18:10:56] [Debug]    Rebroadcasting announce for <55c16056b1b0f93042c92ea31ceb801a> with hop count 16
[2026-04-13 18:10:56] [Debug]    Rebroadcasting announce for <110d7f3159c1d306851c3ec5c6d302ef> with hop count 4
[2026-04-13 18:10:56] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 4
[2026-04-13 18:10:56] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 9.08ms
[2026-04-13 18:10:56] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.63ms
[2026-04-13 18:10:56] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.39ms
[2026-04-13 18:10:56] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.18ms
[2026-04-13 18:10:56] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.14ms
[2026-04-13 18:10:56] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.11ms
[2026-04-13 18:10:56] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.04ms
[2026-04-13 18:10:56] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.01ms
[2026-04-13 18:10:56] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.97ms
[2026-04-13 18:10:56] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.91ms
[2026-04-13 18:10:56] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.87ms
[2026-04-13 18:10:56] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.84ms
[2026-04-13 18:10:56] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:56] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:56] [Extra]    Valid announce for <af1ec9121da534836e6a39b7d261fa65> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:56] [Debug]    Destination <af1ec9121da534836e6a39b7d261fa65> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:57] [Extra]    Completed announce processing for <424157ea5e1bb6dd1aa6dba8970460cf>, local rebroadcast limit reached
[2026-04-13 18:10:57] [Extra]    Completed announce processing for <29b2ebe588859e48aabf13e97cfe245b>, local rebroadcast limit reached
[2026-04-13 18:10:57] [Debug]    Rebroadcasting announce for <af1ec9121da534836e6a39b7d261fa65> with hop count 3
[2026-04-13 18:10:57] [Extra]    Valid announce for <af1ec9121da534836e6a39b7d261fa65> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:57] [Extra]    Heard a rebroadcast of announce for <af1ec9121da534836e6a39b7d261fa65> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:57] [Extra]    Valid announce for <3b171e0b79acf468ae1bf3a6d8515d12> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:57] [Debug]    Destination <3b171e0b79acf468ae1bf3a6d8515d12> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:58] [Debug]    Rebroadcasting announce for <9eb0f17e1691e819bb34eddafa2ca82b> with hop count 3
[2026-04-13 18:10:58] [Debug]    Rebroadcasting announce for <e345f6220682e127cab52c3387436778> with hop count 5
[2026-04-13 18:10:58] [Debug]    Rebroadcasting announce for <73400f494c8d580bd774443a5163127b> with hop count 3
[2026-04-13 18:10:58] [Debug]    Rebroadcasting announce for <02aaf088472435718061211d3752c8ed> with hop count 4
[2026-04-13 18:10:58] [Debug]    Rebroadcasting announce for <f202ae6541f5e69c204d0b2bcbfcd273> with hop count 4
[2026-04-13 18:10:58] [Debug]    Rebroadcasting announce for <103eb3c7f35278ba33e7d014e341b3ec> with hop count 4
[2026-04-13 18:10:58] [Debug]    Rebroadcasting announce for <ca273d664d1a6c59a5a002670a641eff> with hop count 3
[2026-04-13 18:10:58] [Debug]    Rebroadcasting announce for <3b171e0b79acf468ae1bf3a6d8515d12> with hop count 4
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.28ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.98ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.85ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.66ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.64ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.62ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.59ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.57ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.56ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.53ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.51ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.49ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.46ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.45ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.43ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.4ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.39ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.37ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.34ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.33ms
[2026-04-13 18:10:58] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.31ms
[2026-04-13 18:10:58] [Extra]    Valid announce for <3b171e0b79acf468ae1bf3a6d8515d12> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:58] [Extra]    Heard a rebroadcast of announce for <3b171e0b79acf468ae1bf3a6d8515d12> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:10:58] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:58] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:10:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:58] [Extra]    Valid announce for <935168fd3d4582a802036c6958010662> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:58] [Debug]    Destination <935168fd3d4582a802036c6958010662> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:58] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-13 18:10:58] [Warning]  Attempting to reconnect...
[2026-04-13 18:10:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 18:10:58] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:10:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:10:58] [Extra]    Valid announce for <935168fd3d4582a802036c6958010662> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:58] [Extra]    Valid announce for <4fe37a4e22f312f89f23f50d0ae30185> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:58] [Debug]    Destination <4fe37a4e22f312f89f23f50d0ae30185> is now 5 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:10:59] [Extra]    Completed announce processing for <9eb0f17e1691e819bb34eddafa2ca82b>, local rebroadcast limit reached
[2026-04-13 18:10:59] [Extra]    Completed announce processing for <e345f6220682e127cab52c3387436778>, local rebroadcast limit reached
[2026-04-13 18:10:59] [Extra]    Completed announce processing for <73400f494c8d580bd774443a5163127b>, local rebroadcast limit reached
[2026-04-13 18:10:59] [Extra]    Completed announce processing for <02aaf088472435718061211d3752c8ed>, local rebroadcast limit reached
[2026-04-13 18:10:59] [Extra]    Completed announce processing for <f202ae6541f5e69c204d0b2bcbfcd273>, local rebroadcast limit reached
[2026-04-13 18:10:59] [Extra]    Completed announce processing for <103eb3c7f35278ba33e7d014e341b3ec>, local rebroadcast limit reached
[2026-04-13 18:10:59] [Extra]    Completed announce processing for <ca273d664d1a6c59a5a002670a641eff>, local rebroadcast limit reached
[2026-04-13 18:10:59] [Debug]    Rebroadcasting announce for <fb31704d247f8b1547f9a3b356760e7b> with hop count 10
[2026-04-13 18:10:59] [Debug]    Rebroadcasting announce for <dc085bf490f24f9051a492155bf5e283> with hop count 8
[2026-04-13 18:10:59] [Debug]    Rebroadcasting announce for <54f0e7796ac804890832cb3ee61131f2> with hop count 4
[2026-04-13 18:10:59] [Debug]    Rebroadcasting announce for <dce1135780d75f28804a92545aea418a> with hop count 4
[2026-04-13 18:10:59] [Debug]    Rebroadcasting announce for <12cb1ed29943213839f0b0d18cd42761> with hop count 4
[2026-04-13 18:10:59] [Debug]    Rebroadcasting announce for <935168fd3d4582a802036c6958010662> with hop count 3
[2026-04-13 18:10:59] [Debug]    Rebroadcasting announce for <4fe37a4e22f312f89f23f50d0ae30185> with hop count 5
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.61ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.4ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.3ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.18ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.15ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.1ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.07ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.06ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.04ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.02ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.01ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.99ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.97ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.95ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.94ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.88ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.85ms
[2026-04-13 18:10:59] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.84ms
[2026-04-13 18:10:59] [Extra]    Valid announce for <935168fd3d4582a802036c6958010662> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:59] [Extra]    Heard a rebroadcast of announce for <935168fd3d4582a802036c6958010662> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:59] [Extra]    Valid announce for <4fe37a4e22f312f89f23f50d0ae30185> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:59] [Extra]    Heard a rebroadcast of announce for <4fe37a4e22f312f89f23f50d0ae30185> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:10:59] [Extra]    Valid announce for <8bbd266fcef9a841fcd8c1cab0bc0388> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:59] [Extra]    Remembering ratchet <c8580e95b64a828dc0f3> for <8bbd266fcef9a841fcd8c1cab0bc0388>
[2026-04-13 18:10:59] [Debug]    Destination <8bbd266fcef9a841fcd8c1cab0bc0388> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:10:59] [Extra]    Valid announce for <8bbd266fcef9a841fcd8c1cab0bc0388> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:00] [Extra]    Completed announce processing for <fb31704d247f8b1547f9a3b356760e7b>, local rebroadcast limit reached
[2026-04-13 18:11:00] [Extra]    Completed announce processing for <dc085bf490f24f9051a492155bf5e283>, local rebroadcast limit reached
[2026-04-13 18:11:00] [Extra]    Completed announce processing for <54f0e7796ac804890832cb3ee61131f2>, local rebroadcast limit reached
[2026-04-13 18:11:00] [Extra]    Completed announce processing for <dce1135780d75f28804a92545aea418a>, local rebroadcast limit reached
[2026-04-13 18:11:00] [Extra]    Completed announce processing for <12cb1ed29943213839f0b0d18cd42761>, local rebroadcast limit reached
[2026-04-13 18:11:00] [Debug]    Rebroadcasting announce for <768fb269391fb64afc979dffe6572cb2> with hop count 13
[2026-04-13 18:11:00] [Debug]    Rebroadcasting announce for <587a35e51879b1954788cca5c3be2ff7> with hop count 11
[2026-04-13 18:11:00] [Debug]    Rebroadcasting announce for <34593130ea32ad163dff5b3066c43964> with hop count 5
[2026-04-13 18:11:00] [Debug]    Rebroadcasting announce for <4dd9ed5f1d35aa5e95f9e34477bef74d> with hop count 3
[2026-04-13 18:11:00] [Debug]    Rebroadcasting announce for <8bbd266fcef9a841fcd8c1cab0bc0388> with hop count 3
[2026-04-13 18:11:00] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.03ms
[2026-04-13 18:11:00] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.8ms
[2026-04-13 18:11:00] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.66ms
[2026-04-13 18:11:00] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.5ms
[2026-04-13 18:11:00] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.48ms
[2026-04-13 18:11:00] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.46ms
[2026-04-13 18:11:00] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.33ms
[2026-04-13 18:11:00] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.31ms
[2026-04-13 18:11:00] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.29ms
[2026-04-13 18:11:00] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.26ms
[2026-04-13 18:11:00] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.24ms
[2026-04-13 18:11:00] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.22ms
[2026-04-13 18:11:00] [Extra]    Valid announce for <46820c17ae36924b2a39f1de43014c7d> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:00] [Debug]    Destination <46820c17ae36924b2a39f1de43014c7d> is now 5 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:01] [Extra]    Completed announce processing for <768fb269391fb64afc979dffe6572cb2>, local rebroadcast limit reached
[2026-04-13 18:11:01] [Extra]    Completed announce processing for <587a35e51879b1954788cca5c3be2ff7>, local rebroadcast limit reached
[2026-04-13 18:11:01] [Extra]    Completed announce processing for <34593130ea32ad163dff5b3066c43964>, local rebroadcast limit reached
[2026-04-13 18:11:01] [Extra]    Completed announce processing for <4dd9ed5f1d35aa5e95f9e34477bef74d>, local rebroadcast limit reached
[2026-04-13 18:11:01] [Debug]    Rebroadcasting announce for <3c81447dff85b425c79ca5a97ff75f75> with hop count 9
[2026-04-13 18:11:01] [Debug]    Rebroadcasting announce for <38073923c15b25893cd38a7938a9943a> with hop count 10
[2026-04-13 18:11:01] [Debug]    Rebroadcasting announce for <794884194914d03c4e199d9c1f090b0c> with hop count 6
[2026-04-13 18:11:01] [Debug]    Rebroadcasting announce for <46820c17ae36924b2a39f1de43014c7d> with hop count 5
[2026-04-13 18:11:01] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.19ms
[2026-04-13 18:11:01] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.87ms
[2026-04-13 18:11:01] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.59ms
[2026-04-13 18:11:01] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.35ms
[2026-04-13 18:11:01] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.31ms
[2026-04-13 18:11:01] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.28ms
[2026-04-13 18:11:01] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.21ms
[2026-04-13 18:11:01] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.18ms
[2026-04-13 18:11:01] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.15ms
[2026-04-13 18:11:01] [Debug]    Path request for <0597ced5c2da350f4df3a506571284ca> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:01] [Debug]    Ignoring path request for <0597ced5c2da350f4df3a506571284ca> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 18:11:02] [Extra]    Completed announce processing for <3c81447dff85b425c79ca5a97ff75f75>, local rebroadcast limit reached
[2026-04-13 18:11:02] [Extra]    Completed announce processing for <38073923c15b25893cd38a7938a9943a>, local rebroadcast limit reached
[2026-04-13 18:11:02] [Extra]    Completed announce processing for <794884194914d03c4e199d9c1f090b0c>, local rebroadcast limit reached
[2026-04-13 18:11:02] [Debug]    Rebroadcasting announce for <55c16056b1b0f93042c92ea31ceb801a> with hop count 16
[2026-04-13 18:11:02] [Debug]    Rebroadcasting announce for <110d7f3159c1d306851c3ec5c6d302ef> with hop count 4
[2026-04-13 18:11:02] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 4
[2026-04-13 18:11:02] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.39ms
[2026-04-13 18:11:02] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.23ms
[2026-04-13 18:11:02] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.94ms
[2026-04-13 18:11:02] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.89ms
[2026-04-13 18:11:02] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.88ms
[2026-04-13 18:11:02] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.87ms
[2026-04-13 18:11:02] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:02] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:02] [Extra]    Valid announce for <284395a82e44594554a03e3757988396> 19 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:02] [Extra]    Remembering ratchet <e1d55e17846a54c5f756> for <284395a82e44594554a03e3757988396>
[2026-04-13 18:11:02] [Debug]    Destination <284395a82e44594554a03e3757988396> is now 19 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:02] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:02] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:02] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:02] [Extra]    Valid announce for <284395a82e44594554a03e3757988396> 19 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:03] [Extra]    Completed announce processing for <55c16056b1b0f93042c92ea31ceb801a>, local rebroadcast limit reached
[2026-04-13 18:11:03] [Extra]    Completed announce processing for <110d7f3159c1d306851c3ec5c6d302ef>, local rebroadcast limit reached
[2026-04-13 18:11:03] [Extra]    Completed announce processing for <2d8a25919ea488ce008d3635d9b104c7>, local rebroadcast limit reached
[2026-04-13 18:11:03] [Debug]    Rebroadcasting announce for <af1ec9121da534836e6a39b7d261fa65> with hop count 3
[2026-04-13 18:11:03] [Debug]    Rebroadcasting announce for <73400f494c8d580bd774443a5163127b> with hop count 3
[2026-04-13 18:11:03] [Debug]    Rebroadcasting announce for <284395a82e44594554a03e3757988396> with hop count 19
[2026-04-13 18:11:03] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.91ms
[2026-04-13 18:11:03] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.57ms
[2026-04-13 18:11:03] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.4ms
[2026-04-13 18:11:03] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.07ms
[2026-04-13 18:11:03] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.03ms
[2026-04-13 18:11:03] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.99ms
[2026-04-13 18:11:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:11:03] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:03] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:11:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:03] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:03] [Extra]    Valid announce for <284395a82e44594554a03e3757988396> 20 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:03] [Extra]    Heard a rebroadcast of announce for <284395a82e44594554a03e3757988396> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 18:11:03] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:03] [Extra]    Valid announce for <6034cb0a2644c5a7c47ccf24f9707f55> 15 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:03] [Extra]    Remembering ratchet <e80d9c5f8cf76dac64d5> for <6034cb0a2644c5a7c47ccf24f9707f55>
[2026-04-13 18:11:03] [Debug]    Destination <6034cb0a2644c5a7c47ccf24f9707f55> is now 15 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:03] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-13 18:11:03] [Warning]  Attempting to reconnect...
[2026-04-13 18:11:03] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:03] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:03] [Extra]    Valid announce for <a430b813dd5c253002380cda46bf8a05> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:03] [Extra]    Remembering ratchet <26c08214ae28bdc493cc> for <a430b813dd5c253002380cda46bf8a05>
[2026-04-13 18:11:03] [Debug]    Destination <a430b813dd5c253002380cda46bf8a05> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:04] [Extra]    Completed announce processing for <af1ec9121da534836e6a39b7d261fa65>, local rebroadcast limit reached
[2026-04-13 18:11:04] [Debug]    Rebroadcasting announce for <3b171e0b79acf468ae1bf3a6d8515d12> with hop count 4
[2026-04-13 18:11:04] [Debug]    Rebroadcasting announce for <ca273d664d1a6c59a5a002670a641eff> with hop count 3
[2026-04-13 18:11:04] [Debug]    Rebroadcasting announce for <6034cb0a2644c5a7c47ccf24f9707f55> with hop count 15
[2026-04-13 18:11:04] [Debug]    Rebroadcasting announce for <110d7f3159c1d306851c3ec5c6d302ef> with hop count 4
[2026-04-13 18:11:04] [Debug]    Rebroadcasting announce for <a430b813dd5c253002380cda46bf8a05> with hop count 4
[2026-04-13 18:11:04] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.78ms
[2026-04-13 18:11:04] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.64ms
[2026-04-13 18:11:04] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.56ms
[2026-04-13 18:11:04] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.46ms
[2026-04-13 18:11:04] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.45ms
[2026-04-13 18:11:04] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.44ms
[2026-04-13 18:11:04] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.42ms
[2026-04-13 18:11:04] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.42ms
[2026-04-13 18:11:04] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.41ms
[2026-04-13 18:11:04] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.4ms
[2026-04-13 18:11:04] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.39ms
[2026-04-13 18:11:04] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.39ms
[2026-04-13 18:11:04] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:04] [Extra]    Valid announce for <a430b813dd5c253002380cda46bf8a05> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:04] [Extra]    Heard a rebroadcast of announce for <a430b813dd5c253002380cda46bf8a05> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:04] [Extra]    Valid announce for <6862f26ba0bd11ecb058676e99192762> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:04] [Debug]    Replacing destination table entry for <6862f26ba0bd11ecb058676e99192762> with new announce, since it was more recently emitted
[2026-04-13 18:11:04] [Debug]    Destination <6862f26ba0bd11ecb058676e99192762> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:04] [Extra]    Valid announce for <31d518e10003a12ff62fe56f3484dabe> 13 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:04] [Debug]    Destination <31d518e10003a12ff62fe56f3484dabe> is now 13 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:04] [Extra]    Valid announce for <aea9a5706da57d932ffa95cf58d0bac8> 10 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:04] [Extra]    Remembering ratchet <35608dfea8d04db35f97> for <aea9a5706da57d932ffa95cf58d0bac8>
[2026-04-13 18:11:04] [Debug]    Destination <aea9a5706da57d932ffa95cf58d0bac8> is now 10 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:04] [Extra]    Valid announce for <6862f26ba0bd11ecb058676e99192762> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:05] [Extra]    Completed announce processing for <3b171e0b79acf468ae1bf3a6d8515d12>, local rebroadcast limit reached
[2026-04-13 18:11:05] [Debug]    Rebroadcasting announce for <935168fd3d4582a802036c6958010662> with hop count 3
[2026-04-13 18:11:05] [Debug]    Rebroadcasting announce for <4fe37a4e22f312f89f23f50d0ae30185> with hop count 5
[2026-04-13 18:11:05] [Debug]    Rebroadcasting announce for <6862f26ba0bd11ecb058676e99192762> with hop count 5
[2026-04-13 18:11:05] [Debug]    Rebroadcasting announce for <31d518e10003a12ff62fe56f3484dabe> with hop count 13
[2026-04-13 18:11:05] [Debug]    Rebroadcasting announce for <aea9a5706da57d932ffa95cf58d0bac8> with hop count 10
[2026-04-13 18:11:05] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.81ms
[2026-04-13 18:11:05] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.43ms
[2026-04-13 18:11:05] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.17ms
[2026-04-13 18:11:05] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.89ms
[2026-04-13 18:11:05] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.85ms
[2026-04-13 18:11:05] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.81ms
[2026-04-13 18:11:05] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.74ms
[2026-04-13 18:11:05] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.71ms
[2026-04-13 18:11:05] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.68ms
[2026-04-13 18:11:05] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.61ms
[2026-04-13 18:11:05] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.58ms
[2026-04-13 18:11:05] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.55ms
[2026-04-13 18:11:05] [Debug]    Path request for <1ba3f953b8e28bd2f5a5ec2e741edf65> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:05] [Debug]    Answering path request for <1ba3f953b8e28bd2f5a5ec2e741edf65> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], path is known
[2026-04-13 18:11:06] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:06] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:06] [Extra]    Completed announce processing for <935168fd3d4582a802036c6958010662>, local rebroadcast limit reached
[2026-04-13 18:11:06] [Extra]    Completed announce processing for <4fe37a4e22f312f89f23f50d0ae30185>, local rebroadcast limit reached
[2026-04-13 18:11:06] [Debug]    Rebroadcasting announce for <8bbd266fcef9a841fcd8c1cab0bc0388> with hop count 3
[2026-04-13 18:11:06] [Debug]    Rebroadcasting announce as path response for <1ba3f953b8e28bd2f5a5ec2e741edf65> with hop count 2
[2026-04-13 18:11:06] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 4
[2026-04-13 18:11:06] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.6ms
[2026-04-13 18:11:06] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.21ms
[2026-04-13 18:11:06] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.95ms
[2026-04-13 18:11:06] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:06] [Extra]    Heard a rebroadcast of announce for <2d8a25919ea488ce008d3635d9b104c7> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:07] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:07] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:07] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:07] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:07] [Extra]    Completed announce processing for <8bbd266fcef9a841fcd8c1cab0bc0388>, local rebroadcast limit reached
[2026-04-13 18:11:07] [Debug]    Rebroadcasting announce for <46820c17ae36924b2a39f1de43014c7d> with hop count 5
[2026-04-13 18:11:07] [Extra]    Completed announce processing for <1ba3f953b8e28bd2f5a5ec2e741edf65>, local rebroadcast limit reached
[2026-04-13 18:11:07] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:07] [Debug]    Path request for <995cc3851347f138f51af73d182d5e1d> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:07] [Debug]    Ignoring path request for <995cc3851347f138f51af73d182d5e1d> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 18:11:07] [Extra]    Valid announce for <125bfe4fc66ac35f73e30c2613ab09fa> 8 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:07] [Debug]    Destination <125bfe4fc66ac35f73e30c2613ab09fa> is now 8 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:08] [Extra]    Valid announce for <125bfe4fc66ac35f73e30c2613ab09fa> 8 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:08] [Extra]    Completed announce processing for <46820c17ae36924b2a39f1de43014c7d>, local rebroadcast limit reached
[2026-04-13 18:11:08] [Debug]    Rebroadcasting announce for <73400f494c8d580bd774443a5163127b> with hop count 3
[2026-04-13 18:11:08] [Debug]    Rebroadcasting announce for <ca273d664d1a6c59a5a002670a641eff> with hop count 3
[2026-04-13 18:11:08] [Debug]    Rebroadcasting announce for <125bfe4fc66ac35f73e30c2613ab09fa> with hop count 8
[2026-04-13 18:11:08] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.87ms
[2026-04-13 18:11:08] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.56ms
[2026-04-13 18:11:08] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.44ms
[2026-04-13 18:11:08] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.29ms
[2026-04-13 18:11:08] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.26ms
[2026-04-13 18:11:08] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.25ms
[2026-04-13 18:11:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:11:08] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:08] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:11:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:08] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-13 18:11:08] [Warning]  Attempting to reconnect...
[2026-04-13 18:11:08] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 18:11:08] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:08] [Extra]    Valid announce for <422d3e345b5e84f843ade3821f28cf9e> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:08] [Debug]    Destination <422d3e345b5e84f843ade3821f28cf9e> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:09] [Extra]    Valid announce for <3b171e0b79acf468ae1bf3a6d8515d12> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:09] [Debug]    Destination <3b171e0b79acf468ae1bf3a6d8515d12> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:09] [Extra]    Valid announce for <ef98bc2981f7d1e793bb391ca0208231> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:09] [Debug]    Destination <ef98bc2981f7d1e793bb391ca0208231> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:09] [Debug]    Rebroadcasting announce for <284395a82e44594554a03e3757988396> with hop count 19
[2026-04-13 18:11:09] [Debug]    Rebroadcasting announce for <422d3e345b5e84f843ade3821f28cf9e> with hop count 4
[2026-04-13 18:11:09] [Debug]    Rebroadcasting announce for <ef98bc2981f7d1e793bb391ca0208231> with hop count 4
[2026-04-13 18:11:09] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.63ms
[2026-04-13 18:11:09] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.32ms
[2026-04-13 18:11:09] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.86ms
[2026-04-13 18:11:09] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.49ms
[2026-04-13 18:11:09] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.46ms
[2026-04-13 18:11:09] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.43ms
[2026-04-13 18:11:09] [Extra]    Valid announce for <3b171e0b79acf468ae1bf3a6d8515d12> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:09] [Extra]    Heard a rebroadcast of announce for <3b171e0b79acf468ae1bf3a6d8515d12> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:09] [Extra]    Valid announce for <ef98bc2981f7d1e793bb391ca0208231> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:09] [Extra]    Heard a rebroadcast of announce for <ef98bc2981f7d1e793bb391ca0208231> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:09] [Extra]    Valid announce for <7d53d87143d698fc327d8a31f9a54751> 10 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:09] [Debug]    Destination <7d53d87143d698fc327d8a31f9a54751> is now 10 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:10] [Extra]    Completed announce processing for <284395a82e44594554a03e3757988396>, local rebroadcast limit reached
[2026-04-13 18:11:10] [Debug]    Rebroadcasting announce for <6034cb0a2644c5a7c47ccf24f9707f55> with hop count 15
[2026-04-13 18:11:10] [Debug]    Rebroadcasting announce for <110d7f3159c1d306851c3ec5c6d302ef> with hop count 4
[2026-04-13 18:11:10] [Debug]    Rebroadcasting announce for <a430b813dd5c253002380cda46bf8a05> with hop count 4
[2026-04-13 18:11:10] [Debug]    Rebroadcasting announce for <3b171e0b79acf468ae1bf3a6d8515d12> with hop count 4
[2026-04-13 18:11:10] [Debug]    Rebroadcasting announce for <7d53d87143d698fc327d8a31f9a54751> with hop count 10
[2026-04-13 18:11:10] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 9.07ms
[2026-04-13 18:11:10] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.62ms
[2026-04-13 18:11:10] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.48ms
[2026-04-13 18:11:10] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.35ms
[2026-04-13 18:11:10] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.33ms
[2026-04-13 18:11:10] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.32ms
[2026-04-13 18:11:10] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.21ms
[2026-04-13 18:11:10] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.2ms
[2026-04-13 18:11:10] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.19ms
[2026-04-13 18:11:10] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.16ms
[2026-04-13 18:11:10] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.15ms
[2026-04-13 18:11:10] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.14ms
[2026-04-13 18:11:10] [Extra]    Valid announce for <7d53d87143d698fc327d8a31f9a54751> 10 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:10] [Extra]    Valid announce for <6ea9ba935598917da4dd305962c2760c> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:10] [Debug]    Destination <6ea9ba935598917da4dd305962c2760c> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:10] [Extra]    Valid announce for <f0921573abf41194ce99f3976f3c7792> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:10] [Debug]    Destination <f0921573abf41194ce99f3976f3c7792> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:10] [Extra]    Valid announce for <f0921573abf41194ce99f3976f3c7792> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:10] [Extra]    Heard a rebroadcast of announce for <f0921573abf41194ce99f3976f3c7792> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:11] [Extra]    Valid announce for <6ea9ba935598917da4dd305962c2760c> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:11] [Extra]    Heard a rebroadcast of announce for <6ea9ba935598917da4dd305962c2760c> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:11] [Extra]    Valid announce for <f0921573abf41194ce99f3976f3c7792> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:11] [Extra]    Heard a rebroadcast of announce for <f0921573abf41194ce99f3976f3c7792> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:11] [Extra]    Completed announce processing for <6034cb0a2644c5a7c47ccf24f9707f55>, local rebroadcast limit reached
[2026-04-13 18:11:11] [Extra]    Completed announce processing for <110d7f3159c1d306851c3ec5c6d302ef>, local rebroadcast limit reached
[2026-04-13 18:11:11] [Extra]    Completed announce processing for <a430b813dd5c253002380cda46bf8a05>, local rebroadcast limit reached
[2026-04-13 18:11:11] [Debug]    Rebroadcasting announce for <6862f26ba0bd11ecb058676e99192762> with hop count 5
[2026-04-13 18:11:11] [Debug]    Rebroadcasting announce for <31d518e10003a12ff62fe56f3484dabe> with hop count 13
[2026-04-13 18:11:11] [Debug]    Rebroadcasting announce for <aea9a5706da57d932ffa95cf58d0bac8> with hop count 10
[2026-04-13 18:11:11] [Debug]    Rebroadcasting announce for <6ea9ba935598917da4dd305962c2760c> with hop count 2
[2026-04-13 18:11:11] [Debug]    Rebroadcasting announce for <f0921573abf41194ce99f3976f3c7792> with hop count 2
[2026-04-13 18:11:11] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.08ms
[2026-04-13 18:11:11] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.72ms
[2026-04-13 18:11:11] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.4ms
[2026-04-13 18:11:11] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.12ms
[2026-04-13 18:11:11] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.07ms
[2026-04-13 18:11:11] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.04ms
[2026-04-13 18:11:11] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.97ms
[2026-04-13 18:11:11] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.94ms
[2026-04-13 18:11:11] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.91ms
[2026-04-13 18:11:11] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.84ms
[2026-04-13 18:11:11] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.81ms
[2026-04-13 18:11:11] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.78ms
[2026-04-13 18:11:11] [Extra]    Valid announce for <6ea9ba935598917da4dd305962c2760c> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:11] [Extra]    Heard a rebroadcast of announce for <6ea9ba935598917da4dd305962c2760c> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:11] [Extra]    Completed announce processing for <6ea9ba935598917da4dd305962c2760c>, local rebroadcast limit reached
[2026-04-13 18:11:11] [Extra]    Valid announce for <7a9d8bc724b1c75fa75da91c433886e9> 13 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:11] [Debug]    Destination <7a9d8bc724b1c75fa75da91c433886e9> is now 13 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:12] [Extra]    Valid announce for <44213278b52d888683e970004dc95f3c> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:12] [Debug]    Destination <44213278b52d888683e970004dc95f3c> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:12] [Extra]    Completed announce processing for <6862f26ba0bd11ecb058676e99192762>, local rebroadcast limit reached
[2026-04-13 18:11:12] [Extra]    Completed announce processing for <31d518e10003a12ff62fe56f3484dabe>, local rebroadcast limit reached
[2026-04-13 18:11:12] [Extra]    Completed announce processing for <aea9a5706da57d932ffa95cf58d0bac8>, local rebroadcast limit reached
[2026-04-13 18:11:12] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 4
[2026-04-13 18:11:12] [Debug]    Rebroadcasting announce for <7a9d8bc724b1c75fa75da91c433886e9> with hop count 13
[2026-04-13 18:11:12] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.93ms
[2026-04-13 18:11:12] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.65ms
[2026-04-13 18:11:12] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.34ms
[2026-04-13 18:11:12] [Extra]    Valid announce for <44213278b52d888683e970004dc95f3c> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:12] [Extra]    Heard a rebroadcast of announce for <44213278b52d888683e970004dc95f3c> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:11:13] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:13] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:11:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:13] [Extra]    Completed announce processing for <2d8a25919ea488ce008d3635d9b104c7>, local rebroadcast limit reached
[2026-04-13 18:11:13] [Debug]    Rebroadcasting announce for <44213278b52d888683e970004dc95f3c> with hop count 4
[2026-04-13 18:11:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 18:11:13] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:13] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-13 18:11:13] [Warning]  Attempting to reconnect...
[2026-04-13 18:11:13] [Extra]    Valid announce for <0040730e1a189246b832aca13b68de2b> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:13] [Extra]    Remembering ratchet <5a11d8ac40c6ad608e4c> for <0040730e1a189246b832aca13b68de2b>
[2026-04-13 18:11:13] [Debug]    Destination <0040730e1a189246b832aca13b68de2b> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:14] [Debug]    Rebroadcasting announce for <73400f494c8d580bd774443a5163127b> with hop count 3
[2026-04-13 18:11:14] [Debug]    Rebroadcasting announce for <ca273d664d1a6c59a5a002670a641eff> with hop count 3
[2026-04-13 18:11:14] [Debug]    Rebroadcasting announce for <125bfe4fc66ac35f73e30c2613ab09fa> with hop count 8
[2026-04-13 18:11:14] [Debug]    Rebroadcasting announce for <0040730e1a189246b832aca13b68de2b> with hop count 4
[2026-04-13 18:11:14] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.91ms
[2026-04-13 18:11:14] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.59ms
[2026-04-13 18:11:14] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.44ms
[2026-04-13 18:11:14] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.28ms
[2026-04-13 18:11:14] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.26ms
[2026-04-13 18:11:14] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.24ms
[2026-04-13 18:11:14] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.21ms
[2026-04-13 18:11:14] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.19ms
[2026-04-13 18:11:14] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.17ms
[2026-04-13 18:11:14] [Extra]    Valid announce for <0040730e1a189246b832aca13b68de2b> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:15] [Extra]    Completed announce processing for <73400f494c8d580bd774443a5163127b>, local rebroadcast limit reached
[2026-04-13 18:11:15] [Extra]    Completed announce processing for <ca273d664d1a6c59a5a002670a641eff>, local rebroadcast limit reached
[2026-04-13 18:11:15] [Extra]    Completed announce processing for <125bfe4fc66ac35f73e30c2613ab09fa>, local rebroadcast limit reached
[2026-04-13 18:11:15] [Debug]    Rebroadcasting announce for <422d3e345b5e84f843ade3821f28cf9e> with hop count 4
[2026-04-13 18:11:15] [Debug]    Rebroadcasting announce for <ef98bc2981f7d1e793bb391ca0208231> with hop count 4
[2026-04-13 18:11:15] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.77ms
[2026-04-13 18:11:15] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.4ms
[2026-04-13 18:11:15] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.25ms
[2026-04-13 18:11:16] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:16] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:16] [Extra]    Valid announce for <95ca807f05d258f7723d5f1f75c29159> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:16] [Debug]    Destination <95ca807f05d258f7723d5f1f75c29159> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:16] [Extra]    Completed announce processing for <422d3e345b5e84f843ade3821f28cf9e>, local rebroadcast limit reached
[2026-04-13 18:11:16] [Debug]    Rebroadcasting announce for <3b171e0b79acf468ae1bf3a6d8515d12> with hop count 4
[2026-04-13 18:11:16] [Extra]    Completed announce processing for <ef98bc2981f7d1e793bb391ca0208231>, local rebroadcast limit reached
[2026-04-13 18:11:16] [Debug]    Rebroadcasting announce for <7d53d87143d698fc327d8a31f9a54751> with hop count 10
[2026-04-13 18:11:16] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 4
[2026-04-13 18:11:16] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.7ms
[2026-04-13 18:11:16] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.42ms
[2026-04-13 18:11:16] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.27ms
[2026-04-13 18:11:16] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.12ms
[2026-04-13 18:11:16] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.1ms
[2026-04-13 18:11:16] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.08ms
[2026-04-13 18:11:16] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:16] [Extra]    Valid announce for <95ca807f05d258f7723d5f1f75c29159> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:16] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:16] [Debug]    Destination <192eed7af8e3311445372f2a43cb63ec> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:16] [Extra]    Valid announce for <125bfe4fc66ac35f73e30c2613ab09fa> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:16] [Debug]    Destination <125bfe4fc66ac35f73e30c2613ab09fa> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:16] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:17] [Extra]    Valid announce for <125bfe4fc66ac35f73e30c2613ab09fa> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:17] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:17] [Extra]    Valid announce for <ce32ad052da5f355e968cfae2ecc1cea> 9 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:17] [Extra]    Remembering ratchet <7a42e9c334fe61e12fe9> for <ce32ad052da5f355e968cfae2ecc1cea>
[2026-04-13 18:11:17] [Debug]    Destination <ce32ad052da5f355e968cfae2ecc1cea> is now 9 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:17] [Extra]    Completed announce processing for <3b171e0b79acf468ae1bf3a6d8515d12>, local rebroadcast limit reached
[2026-04-13 18:11:17] [Extra]    Completed announce processing for <7d53d87143d698fc327d8a31f9a54751>, local rebroadcast limit reached
[2026-04-13 18:11:17] [Debug]    Rebroadcasting announce for <f0921573abf41194ce99f3976f3c7792> with hop count 2
[2026-04-13 18:11:17] [Debug]    Rebroadcasting announce for <95ca807f05d258f7723d5f1f75c29159> with hop count 3
[2026-04-13 18:11:17] [Debug]    Rebroadcasting announce for <192eed7af8e3311445372f2a43cb63ec> with hop count 2
[2026-04-13 18:11:17] [Debug]    Rebroadcasting announce for <125bfe4fc66ac35f73e30c2613ab09fa> with hop count 4
[2026-04-13 18:11:17] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.81ms
[2026-04-13 18:11:17] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.45ms
[2026-04-13 18:11:17] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.24ms
[2026-04-13 18:11:17] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.0ms
[2026-04-13 18:11:17] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.96ms
[2026-04-13 18:11:17] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.93ms
[2026-04-13 18:11:17] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.87ms
[2026-04-13 18:11:17] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.83ms
[2026-04-13 18:11:17] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.8ms
[2026-04-13 18:11:17] [Extra]    Valid announce for <125bfe4fc66ac35f73e30c2613ab09fa> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:17] [Extra]    Heard a rebroadcast of announce for <125bfe4fc66ac35f73e30c2613ab09fa> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:17] [Extra]    Valid announce for <ce32ad052da5f355e968cfae2ecc1cea> 9 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:17] [Extra]    Valid announce for <7bc02760008fece9c6c82c7076b2084b> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:17] [Debug]    Destination <7bc02760008fece9c6c82c7076b2084b> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:17] [Extra]    Valid announce for <ffc8d3472451090677fd446837e384ff> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:17] [Debug]    Destination <ffc8d3472451090677fd446837e384ff> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:17] [Extra]    Valid announce for <2a2c53858ac1ef449ff10c402a5e512c> 7 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:17] [Extra]    Valid announce for <ce32ad052da5f355e968cfae2ecc1cea> 9 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:17] [Extra]    Valid announce for <5ccfc2181dc7a99bbacf66c585b7155f> 8 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:17] [Debug]    Destination <5ccfc2181dc7a99bbacf66c585b7155f> is now 8 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:17] [Extra]    Valid announce for <ffc8d3472451090677fd446837e384ff> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:17] [Extra]    Valid announce for <3a9c2c862b4b57eefbc107426a1f9126> 10 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:17] [Debug]    Replacing destination table entry for <3a9c2c862b4b57eefbc107426a1f9126> with new announce, since it was more recently emitted
[2026-04-13 18:11:17] [Debug]    Destination <3a9c2c862b4b57eefbc107426a1f9126> is now 10 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:18] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: [Errno 60] Operation timed out
[2026-04-13 18:11:18] [Warning]  Attempting to reconnect...
[2026-04-13 18:11:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:11:18] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:18] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:11:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:18] [Extra]    Valid announce for <ffc8d3472451090677fd446837e384ff> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:18] [Extra]    Valid announce for <ff4d35444e8b7976ced7be2db7812614> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:18] [Debug]    Destination <ff4d35444e8b7976ced7be2db7812614> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:18] [Extra]    Valid announce for <7bc02760008fece9c6c82c7076b2084b> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:18] [Extra]    Completed announce processing for <f0921573abf41194ce99f3976f3c7792>, local rebroadcast limit reached
[2026-04-13 18:11:18] [Debug]    Rebroadcasting announce for <7a9d8bc724b1c75fa75da91c433886e9> with hop count 13
[2026-04-13 18:11:18] [Debug]    Rebroadcasting announce for <ce32ad052da5f355e968cfae2ecc1cea> with hop count 9
[2026-04-13 18:11:18] [Debug]    Rebroadcasting announce for <7bc02760008fece9c6c82c7076b2084b> with hop count 4
[2026-04-13 18:11:18] [Debug]    Rebroadcasting announce for <ffc8d3472451090677fd446837e384ff> with hop count 5
[2026-04-13 18:11:18] [Debug]    Rebroadcasting announce for <5ccfc2181dc7a99bbacf66c585b7155f> with hop count 8
[2026-04-13 18:11:18] [Debug]    Rebroadcasting announce for <3a9c2c862b4b57eefbc107426a1f9126> with hop count 10
[2026-04-13 18:11:18] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.22ms
[2026-04-13 18:11:18] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.98ms
[2026-04-13 18:11:18] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.86ms
[2026-04-13 18:11:18] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.75ms
[2026-04-13 18:11:18] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.73ms
[2026-04-13 18:11:18] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.72ms
[2026-04-13 18:11:18] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.69ms
[2026-04-13 18:11:18] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.67ms
[2026-04-13 18:11:18] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.66ms
[2026-04-13 18:11:18] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.64ms
[2026-04-13 18:11:18] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.62ms
[2026-04-13 18:11:18] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.61ms
[2026-04-13 18:11:18] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.58ms
[2026-04-13 18:11:18] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.57ms
[2026-04-13 18:11:18] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.56ms
[2026-04-13 18:11:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 18:11:18] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:18] [Extra]    Valid announce for <ff4d35444e8b7976ced7be2db7812614> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:19] [Extra]    Valid announce for <e501ecf33dd6a62392733c5de79e9683> 6 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:19] [Extra]    Remembering ratchet <e829baa63052c17285b1> for <e501ecf33dd6a62392733c5de79e9683>
[2026-04-13 18:11:19] [Debug]    Destination <e501ecf33dd6a62392733c5de79e9683> is now 6 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:19] [Extra]    Valid announce for <2a2c53858ac1ef449ff10c402a5e512c> 9 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:19] [Extra]    Valid announce for <664020dc1c08b8e03c17fe09bc5627b8> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:19] [Debug]    Destination <664020dc1c08b8e03c17fe09bc5627b8> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:19] [Extra]    Valid announce for <e501ecf33dd6a62392733c5de79e9683> 8 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:19] [Extra]    Completed announce processing for <7a9d8bc724b1c75fa75da91c433886e9>, local rebroadcast limit reached
[2026-04-13 18:11:19] [Debug]    Rebroadcasting announce for <44213278b52d888683e970004dc95f3c> with hop count 4
[2026-04-13 18:11:19] [Debug]    Rebroadcasting announce for <ff4d35444e8b7976ced7be2db7812614> with hop count 4
[2026-04-13 18:11:19] [Debug]    Rebroadcasting announce for <e501ecf33dd6a62392733c5de79e9683> with hop count 6
[2026-04-13 18:11:19] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.12ms
[2026-04-13 18:11:19] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.88ms
[2026-04-13 18:11:19] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.79ms
[2026-04-13 18:11:19] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.67ms
[2026-04-13 18:11:19] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.58ms
[2026-04-13 18:11:19] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.55ms
[2026-04-13 18:11:19] [Extra]    Valid announce for <664020dc1c08b8e03c17fe09bc5627b8> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:19] [Extra]    Valid announce for <7d53d87143d698fc327d8a31f9a54751> 9 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:19] [Debug]    Destination <7d53d87143d698fc327d8a31f9a54751> is now 9 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:20] [Extra]    Valid announce for <b831f4805fd51cbdc4b549afbd481898> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:20] [Extra]    Remembering ratchet <07eaddeca1101afbf904> for <b831f4805fd51cbdc4b549afbd481898>
[2026-04-13 18:11:20] [Debug]    Destination <b831f4805fd51cbdc4b549afbd481898> is now 5 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:20] [Extra]    Valid announce for <7d53d87143d698fc327d8a31f9a54751> 9 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:20] [Extra]    Completed announce processing for <44213278b52d888683e970004dc95f3c>, local rebroadcast limit reached
[2026-04-13 18:11:20] [Debug]    Rebroadcasting announce for <0040730e1a189246b832aca13b68de2b> with hop count 4
[2026-04-13 18:11:20] [Debug]    Rebroadcasting announce for <664020dc1c08b8e03c17fe09bc5627b8> with hop count 4
[2026-04-13 18:11:20] [Debug]    Rebroadcasting announce for <7d53d87143d698fc327d8a31f9a54751> with hop count 9
[2026-04-13 18:11:20] [Debug]    Rebroadcasting announce for <b831f4805fd51cbdc4b549afbd481898> with hop count 5
[2026-04-13 18:11:20] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 9.32ms
[2026-04-13 18:11:20] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 9.03ms
[2026-04-13 18:11:20] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.84ms
[2026-04-13 18:11:20] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.71ms
[2026-04-13 18:11:20] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.7ms
[2026-04-13 18:11:20] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.68ms
[2026-04-13 18:11:20] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.66ms
[2026-04-13 18:11:20] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.64ms
[2026-04-13 18:11:20] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.63ms
[2026-04-13 18:11:20] [Extra]    Valid announce for <b831f4805fd51cbdc4b549afbd481898> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:20] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:20] [Debug]    Destination <e345f6220682e127cab52c3387436778> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:20] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:21] [Extra]    Valid announce for <9e4352aef634a634d9335eb137fcc82f> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:21] [Debug]    Destination <9e4352aef634a634d9335eb137fcc82f> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:21] [Extra]    Valid announce for <2a2c53858ac1ef449ff10c402a5e512c> 11 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:21] [Extra]    Valid announce for <d7881baf17ece4f8683923d9b1df6f48> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:21] [Debug]    Destination <d7881baf17ece4f8683923d9b1df6f48> is now 5 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:21] [Extra]    Completed announce processing for <0040730e1a189246b832aca13b68de2b>, local rebroadcast limit reached
[2026-04-13 18:11:21] [Debug]    Rebroadcasting announce for <e345f6220682e127cab52c3387436778> with hop count 5
[2026-04-13 18:11:21] [Debug]    Rebroadcasting announce for <9e4352aef634a634d9335eb137fcc82f> with hop count 4
[2026-04-13 18:11:21] [Debug]    Rebroadcasting announce for <d7881baf17ece4f8683923d9b1df6f48> with hop count 5
[2026-04-13 18:11:21] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.69ms
[2026-04-13 18:11:21] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.42ms
[2026-04-13 18:11:21] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.2ms
[2026-04-13 18:11:21] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.92ms
[2026-04-13 18:11:21] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.88ms
[2026-04-13 18:11:21] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.84ms
[2026-04-13 18:11:21] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:21] [Debug]    Replacing destination table entry for <02aaf088472435718061211d3752c8ed> with new announce, since it was more recently emitted
[2026-04-13 18:11:21] [Debug]    Destination <02aaf088472435718061211d3752c8ed> is now 5 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:21] [Extra]    Valid announce for <d7881baf17ece4f8683923d9b1df6f48> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:21] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:21] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:21] [Extra]    Valid announce for <604b95adfa625746d1c9e0c18d7cef75> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:21] [Debug]    Destination <604b95adfa625746d1c9e0c18d7cef75> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:21] [Extra]    Valid announce for <7bc02760008fece9c6c82c7076b2084b> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:21] [Debug]    Destination <7bc02760008fece9c6c82c7076b2084b> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:21] [Extra]    Valid announce for <1f163087a2d335a87bd8dd4ceaf162f2> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:21] [Debug]    Destination <1f163087a2d335a87bd8dd4ceaf162f2> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:21] [Extra]    Valid announce for <d7881baf17ece4f8683923d9b1df6f48> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:21] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:21] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:21] [Extra]    Valid announce for <604b95adfa625746d1c9e0c18d7cef75> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:21] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:21] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:21] [Extra]    Valid announce for <ff4d35444e8b7976ced7be2db7812614> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:21] [Debug]    Destination <ff4d35444e8b7976ced7be2db7812614> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:22] [Extra]    Valid announce for <d7881baf17ece4f8683923d9b1df6f48> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:22] [Extra]    Valid announce for <604b95adfa625746d1c9e0c18d7cef75> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:22] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:22] [Extra]    Heard a rebroadcast of announce for <ca273d664d1a6c59a5a002670a641eff> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:22] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 4
[2026-04-13 18:11:22] [Debug]    Rebroadcasting announce for <7bc02760008fece9c6c82c7076b2084b> with hop count 4
[2026-04-13 18:11:22] [Debug]    Rebroadcasting announce for <ff4d35444e8b7976ced7be2db7812614> with hop count 4
[2026-04-13 18:11:22] [Debug]    Rebroadcasting announce for <02aaf088472435718061211d3752c8ed> with hop count 5
[2026-04-13 18:11:22] [Debug]    Rebroadcasting announce for <ca273d664d1a6c59a5a002670a641eff> with hop count 3
[2026-04-13 18:11:22] [Debug]    Rebroadcasting announce for <604b95adfa625746d1c9e0c18d7cef75> with hop count 4
[2026-04-13 18:11:22] [Debug]    Rebroadcasting announce for <1f163087a2d335a87bd8dd4ceaf162f2> with hop count 4
[2026-04-13 18:11:22] [Debug]    Rebroadcasting announce for <73400f494c8d580bd774443a5163127b> with hop count 3
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.06ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.83ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.71ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.58ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.56ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.55ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.52ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.51ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.5ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.48ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.46ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.45ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.43ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.41ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.37ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.32ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.27ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.25ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.21ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.18ms
[2026-04-13 18:11:22] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.15ms
[2026-04-13 18:11:22] [Extra]    Valid announce for <7bc02760008fece9c6c82c7076b2084b> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:22] [Extra]    Heard a rebroadcast of announce for <7bc02760008fece9c6c82c7076b2084b> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:22] [Extra]    Valid announce for <1f163087a2d335a87bd8dd4ceaf162f2> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:22] [Extra]    Heard a rebroadcast of announce for <1f163087a2d335a87bd8dd4ceaf162f2> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:22] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:22] [Extra]    Valid announce for <ff4d35444e8b7976ced7be2db7812614> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:22] [Extra]    Valid announce for <aea9a5706da57d932ffa95cf58d0bac8> 12 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:22] [Extra]    Valid announce for <55c16056b1b0f93042c92ea31ceb801a> 18 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:22] [Extra]    Valid announce for <284395a82e44594554a03e3757988396> 22 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:23] [Extra]    Valid announce for <1f163087a2d335a87bd8dd4ceaf162f2> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:23] [Extra]    Heard a rebroadcast of announce for <1f163087a2d335a87bd8dd4ceaf162f2> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:23] [Extra]    Completed announce processing for <1f163087a2d335a87bd8dd4ceaf162f2>, local rebroadcast limit reached
[2026-04-13 18:11:23] [Extra]    Valid announce for <21d2069f44093a87e42c94c3bd5393b9> 13 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:23] [Debug]    Destination <21d2069f44093a87e42c94c3bd5393b9> is now 13 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 18:11:23] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:23] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 18:11:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:11:23] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:23] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:11:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:23] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: [Errno 60] Operation timed out
[2026-04-13 18:11:23] [Warning]  Attempting to reconnect...
[2026-04-13 18:11:23] [Extra]    Completed announce processing for <2d8a25919ea488ce008d3635d9b104c7>, local rebroadcast limit reached
[2026-04-13 18:11:23] [Debug]    Rebroadcasting announce for <95ca807f05d258f7723d5f1f75c29159> with hop count 3
[2026-04-13 18:11:23] [Debug]    Rebroadcasting announce for <192eed7af8e3311445372f2a43cb63ec> with hop count 2
[2026-04-13 18:11:23] [Debug]    Rebroadcasting announce for <125bfe4fc66ac35f73e30c2613ab09fa> with hop count 4
[2026-04-13 18:11:23] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.81ms
[2026-04-13 18:11:23] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.66ms
[2026-04-13 18:11:23] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.59ms
[2026-04-13 18:11:23] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.51ms
[2026-04-13 18:11:23] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.49ms
[2026-04-13 18:11:23] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.48ms
[2026-04-13 18:11:23] [Extra]    Valid announce for <219a60c23a74cf1ede2ee1c56dc790d7> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:23] [Debug]    Destination <219a60c23a74cf1ede2ee1c56dc790d7> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:23] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:23] [Debug]    Destination <794884194914d03c4e199d9c1f090b0c> is now 6 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:23] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:23] [Extra]    Valid announce for <21d2069f44093a87e42c94c3bd5393b9> 14 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:23] [Extra]    Heard a rebroadcast of announce for <21d2069f44093a87e42c94c3bd5393b9> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:23] [Extra]    Valid announce for <219a60c23a74cf1ede2ee1c56dc790d7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:23] [Extra]    Heard a rebroadcast of announce for <219a60c23a74cf1ede2ee1c56dc790d7> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:24] [Extra]    Valid announce for <bb2d98ff84fc6b2ccc7767344343f0f7> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:24] [Debug]    Destination <bb2d98ff84fc6b2ccc7767344343f0f7> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:24] [Extra]    Valid announce for <7263a18daca33d7629db3ea58818a274> 7 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:24] [Extra]    Remembering ratchet <d4d2cf96c982316aebbc> for <7263a18daca33d7629db3ea58818a274>
[2026-04-13 18:11:24] [Debug]    Destination <7263a18daca33d7629db3ea58818a274> is now 7 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:24] [Extra]    Completed announce processing for <95ca807f05d258f7723d5f1f75c29159>, local rebroadcast limit reached
[2026-04-13 18:11:24] [Extra]    Completed announce processing for <192eed7af8e3311445372f2a43cb63ec>, local rebroadcast limit reached
[2026-04-13 18:11:24] [Extra]    Completed announce processing for <125bfe4fc66ac35f73e30c2613ab09fa>, local rebroadcast limit reached
[2026-04-13 18:11:24] [Debug]    Rebroadcasting announce for <ce32ad052da5f355e968cfae2ecc1cea> with hop count 9
[2026-04-13 18:11:24] [Debug]    Rebroadcasting announce for <ffc8d3472451090677fd446837e384ff> with hop count 5
[2026-04-13 18:11:24] [Debug]    Rebroadcasting announce for <5ccfc2181dc7a99bbacf66c585b7155f> with hop count 8
[2026-04-13 18:11:24] [Debug]    Rebroadcasting announce for <3a9c2c862b4b57eefbc107426a1f9126> with hop count 10
[2026-04-13 18:11:24] [Debug]    Rebroadcasting announce for <21d2069f44093a87e42c94c3bd5393b9> with hop count 13
[2026-04-13 18:11:24] [Debug]    Rebroadcasting announce for <219a60c23a74cf1ede2ee1c56dc790d7> with hop count 3
[2026-04-13 18:11:24] [Debug]    Rebroadcasting announce for <794884194914d03c4e199d9c1f090b0c> with hop count 6
[2026-04-13 18:11:24] [Debug]    Rebroadcasting announce for <bb2d98ff84fc6b2ccc7767344343f0f7> with hop count 4
[2026-04-13 18:11:24] [Debug]    Rebroadcasting announce for <7263a18daca33d7629db3ea58818a274> with hop count 7
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.94ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.47ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.21ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.94ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.9ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.86ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.79ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.76ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.7ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.6ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.55ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.51ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.43ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.39ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.36ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.3ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.26ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.22ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.14ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.11ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.08ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 8) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.01ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 8) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.98ms
[2026-04-13 18:11:24] [Extra]    Added announce to queue (height 8) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.95ms
[2026-04-13 18:11:25] [Extra]    Valid announce for <219a60c23a74cf1ede2ee1c56dc790d7> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:25] [Extra]    Heard a rebroadcast of announce for <219a60c23a74cf1ede2ee1c56dc790d7> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:25] [Extra]    Completed announce processing for <219a60c23a74cf1ede2ee1c56dc790d7>, local rebroadcast limit reached
[2026-04-13 18:11:25] [Extra]    Valid announce for <3b98ca3ce4b95e607772de2e359cf1b0> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:25] [Extra]    Remembering ratchet <2e0f9c8ff00f274a31c0> for <3b98ca3ce4b95e607772de2e359cf1b0>
[2026-04-13 18:11:25] [Debug]    Destination <3b98ca3ce4b95e607772de2e359cf1b0> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:25] [Extra]    Valid announce for <eae49b952fed85ac694a6896bda42e4b> 11 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:25] [Extra]    Remembering ratchet <b70c54771d764c737d18> for <eae49b952fed85ac694a6896bda42e4b>
[2026-04-13 18:11:25] [Debug]    Destination <eae49b952fed85ac694a6896bda42e4b> is now 11 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:25] [Extra]    Valid announce for <0eae1f0c3d08cb59e08fd9e0c6d978d7> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:25] [Debug]    Destination <0eae1f0c3d08cb59e08fd9e0c6d978d7> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:25] [Extra]    Completed announce processing for <ce32ad052da5f355e968cfae2ecc1cea>, local rebroadcast limit reached
[2026-04-13 18:11:25] [Extra]    Completed announce processing for <ffc8d3472451090677fd446837e384ff>, local rebroadcast limit reached
[2026-04-13 18:11:25] [Extra]    Completed announce processing for <5ccfc2181dc7a99bbacf66c585b7155f>, local rebroadcast limit reached
[2026-04-13 18:11:25] [Extra]    Completed announce processing for <3a9c2c862b4b57eefbc107426a1f9126>, local rebroadcast limit reached
[2026-04-13 18:11:25] [Debug]    Rebroadcasting announce for <e501ecf33dd6a62392733c5de79e9683> with hop count 6
[2026-04-13 18:11:25] [Debug]    Rebroadcasting announce for <3b98ca3ce4b95e607772de2e359cf1b0> with hop count 4
[2026-04-13 18:11:25] [Debug]    Rebroadcasting announce for <eae49b952fed85ac694a6896bda42e4b> with hop count 11
[2026-04-13 18:11:25] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.45ms
[2026-04-13 18:11:25] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.92ms
[2026-04-13 18:11:25] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.58ms
[2026-04-13 18:11:25] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.27ms
[2026-04-13 18:11:25] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.19ms
[2026-04-13 18:11:25] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.14ms
[2026-04-13 18:11:25] [Extra]    Valid announce for <0eae1f0c3d08cb59e08fd9e0c6d978d7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:25] [Extra]    Heard a rebroadcast of announce for <0eae1f0c3d08cb59e08fd9e0c6d978d7> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:25] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:25] [Debug]    Replacing destination table entry for <2d8a25919ea488ce008d3635d9b104c7> with new announce, since it was more recently emitted
[2026-04-13 18:11:25] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:26] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:26] [Extra]    Valid announce for <d0d54b0bddc0c4e4f8c6590a8d03149b> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:26] [Debug]    Destination <d0d54b0bddc0c4e4f8c6590a8d03149b> is now 5 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:26] [Extra]    Completed announce processing for <e501ecf33dd6a62392733c5de79e9683>, local rebroadcast limit reached
[2026-04-13 18:11:26] [Debug]    Rebroadcasting announce for <664020dc1c08b8e03c17fe09bc5627b8> with hop count 4
[2026-04-13 18:11:26] [Debug]    Rebroadcasting announce for <7d53d87143d698fc327d8a31f9a54751> with hop count 9
[2026-04-13 18:11:26] [Debug]    Rebroadcasting announce for <b831f4805fd51cbdc4b549afbd481898> with hop count 5
[2026-04-13 18:11:26] [Debug]    Rebroadcasting announce for <0eae1f0c3d08cb59e08fd9e0c6d978d7> with hop count 4
[2026-04-13 18:11:26] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 5
[2026-04-13 18:11:26] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.33ms
[2026-04-13 18:11:26] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.85ms
[2026-04-13 18:11:26] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.57ms
[2026-04-13 18:11:26] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.24ms
[2026-04-13 18:11:26] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.2ms
[2026-04-13 18:11:26] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.17ms
[2026-04-13 18:11:26] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.1ms
[2026-04-13 18:11:26] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.06ms
[2026-04-13 18:11:26] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.03ms
[2026-04-13 18:11:26] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.96ms
[2026-04-13 18:11:26] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.93ms
[2026-04-13 18:11:26] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.9ms
[2026-04-13 18:11:27] [Extra]    Valid announce for <ba2780f844f711525924923e9bfb23cb> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:27] [Debug]    Destination <ba2780f844f711525924923e9bfb23cb> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:27] [Extra]    Valid announce for <bab9607fc1b1afbc73bd9b455bf3ee98> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:27] [Extra]    Remembering ratchet <b4dfe8369ed8b0a46098> for <bab9607fc1b1afbc73bd9b455bf3ee98>
[2026-04-13 18:11:27] [Debug]    Destination <bab9607fc1b1afbc73bd9b455bf3ee98> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:27] [Extra]    Completed announce processing for <664020dc1c08b8e03c17fe09bc5627b8>, local rebroadcast limit reached
[2026-04-13 18:11:27] [Extra]    Completed announce processing for <7d53d87143d698fc327d8a31f9a54751>, local rebroadcast limit reached
[2026-04-13 18:11:27] [Extra]    Completed announce processing for <b831f4805fd51cbdc4b549afbd481898>, local rebroadcast limit reached
[2026-04-13 18:11:27] [Debug]    Rebroadcasting announce for <e345f6220682e127cab52c3387436778> with hop count 5
[2026-04-13 18:11:27] [Debug]    Rebroadcasting announce for <9e4352aef634a634d9335eb137fcc82f> with hop count 4
[2026-04-13 18:11:27] [Debug]    Rebroadcasting announce for <d7881baf17ece4f8683923d9b1df6f48> with hop count 5
[2026-04-13 18:11:27] [Debug]    Rebroadcasting announce for <d0d54b0bddc0c4e4f8c6590a8d03149b> with hop count 5
[2026-04-13 18:11:27] [Debug]    Rebroadcasting announce for <ba2780f844f711525924923e9bfb23cb> with hop count 3
[2026-04-13 18:11:27] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.51ms
[2026-04-13 18:11:27] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.19ms
[2026-04-13 18:11:27] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.97ms
[2026-04-13 18:11:27] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.76ms
[2026-04-13 18:11:27] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.72ms
[2026-04-13 18:11:27] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.69ms
[2026-04-13 18:11:27] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.61ms
[2026-04-13 18:11:27] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.58ms
[2026-04-13 18:11:27] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.55ms
[2026-04-13 18:11:27] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.49ms
[2026-04-13 18:11:27] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.45ms
[2026-04-13 18:11:27] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.42ms
[2026-04-13 18:11:27] [Extra]    Valid announce for <ba2780f844f711525924923e9bfb23cb> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:27] [Extra]    Heard a rebroadcast of announce for <ba2780f844f711525924923e9bfb23cb> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:27] [Extra]    Valid announce for <bab9607fc1b1afbc73bd9b455bf3ee98> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:27] [Extra]    Heard a rebroadcast of announce for <bab9607fc1b1afbc73bd9b455bf3ee98> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:27] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:27] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:27] [Extra]    Valid announce for <6ee8d89ae74833c397169c07b81e62e2> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:27] [Debug]    Destination <6ee8d89ae74833c397169c07b81e62e2> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:28] [Extra]    Valid announce for <3e9008b06eb94497de65274af72143df> 12 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:28] [Extra]    Remembering ratchet <4c893747ec52ecc0f326> for <3e9008b06eb94497de65274af72143df>
[2026-04-13 18:11:28] [Debug]    Destination <3e9008b06eb94497de65274af72143df> is now 12 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:28] [Extra]    Valid announce for <6ee8d89ae74833c397169c07b81e62e2> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:28] [Extra]    Heard a rebroadcast of announce for <6ee8d89ae74833c397169c07b81e62e2> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:28] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 18:11:28] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:11:28] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:28] [Extra]    Valid announce for <bab9607fc1b1afbc73bd9b455bf3ee98> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:28] [Extra]    Heard a rebroadcast of announce for <bab9607fc1b1afbc73bd9b455bf3ee98> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:28] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:11:28] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 18:11:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:28] [Extra]    Valid announce for <6ee8d89ae74833c397169c07b81e62e2> 2 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 18:11:28] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:28] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 18:11:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:28] [Debug]    Rebroadcasting announce for <7bc02760008fece9c6c82c7076b2084b> with hop count 4
[2026-04-13 18:11:28] [Debug]    Rebroadcasting announce for <ff4d35444e8b7976ced7be2db7812614> with hop count 4
[2026-04-13 18:11:28] [Extra]    Completed announce processing for <e345f6220682e127cab52c3387436778>, local rebroadcast limit reached
[2026-04-13 18:11:28] [Extra]    Completed announce processing for <9e4352aef634a634d9335eb137fcc82f>, local rebroadcast limit reached
[2026-04-13 18:11:28] [Extra]    Completed announce processing for <d7881baf17ece4f8683923d9b1df6f48>, local rebroadcast limit reached
[2026-04-13 18:11:28] [Debug]    Rebroadcasting announce for <02aaf088472435718061211d3752c8ed> with hop count 5
[2026-04-13 18:11:28] [Debug]    Rebroadcasting announce for <ca273d664d1a6c59a5a002670a641eff> with hop count 3
[2026-04-13 18:11:28] [Debug]    Rebroadcasting announce for <604b95adfa625746d1c9e0c18d7cef75> with hop count 4
[2026-04-13 18:11:28] [Debug]    Rebroadcasting announce for <73400f494c8d580bd774443a5163127b> with hop count 3
[2026-04-13 18:11:28] [Debug]    Rebroadcasting announce for <bab9607fc1b1afbc73bd9b455bf3ee98> with hop count 4
[2026-04-13 18:11:28] [Debug]    Rebroadcasting announce for <110d7f3159c1d306851c3ec5c6d302ef> with hop count 4
[2026-04-13 18:11:28] [Debug]    Rebroadcasting announce for <6ee8d89ae74833c397169c07b81e62e2> with hop count 2
[2026-04-13 18:11:28] [Debug]    Rebroadcasting announce for <3e9008b06eb94497de65274af72143df> with hop count 12
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.87ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.51ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.27ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.02ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.98ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.95ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.89ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.85ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.81ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.75ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.72ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.69ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.63ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.43ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.39ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.32ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.29ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.25ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.16ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.13ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.11ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 8) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.08ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 8) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.07ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 8) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.06ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 9) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.01ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 9) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 4.99ms
[2026-04-13 18:11:28] [Extra]    Added announce to queue (height 9) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 4.98ms
[2026-04-13 18:11:28] [Extra]    Valid announce for <3e9008b06eb94497de65274af72143df> 12 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:29] [Extra]    Valid announce for <f90b65ba13050cc114363b47066f4e22> 42 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:29] [Debug]    Destination <f90b65ba13050cc114363b47066f4e22> is now 42 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:29] [Extra]    Valid announce for <9e21f49afdaae8885663e807606387a2> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:29] [Debug]    Destination <9e21f49afdaae8885663e807606387a2> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:29] [Extra]    Valid announce for <9e21f49afdaae8885663e807606387a2> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:29] [Extra]    Valid announce for <0f5a6232102792b2e98c19209070d4f8> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:29] [Debug]    Destination <0f5a6232102792b2e98c19209070d4f8> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:29] [Extra]    Completed announce processing for <7bc02760008fece9c6c82c7076b2084b>, local rebroadcast limit reached
[2026-04-13 18:11:29] [Extra]    Completed announce processing for <ff4d35444e8b7976ced7be2db7812614>, local rebroadcast limit reached
[2026-04-13 18:11:29] [Extra]    Completed announce processing for <02aaf088472435718061211d3752c8ed>, local rebroadcast limit reached
[2026-04-13 18:11:29] [Extra]    Completed announce processing for <ca273d664d1a6c59a5a002670a641eff>, local rebroadcast limit reached
[2026-04-13 18:11:29] [Extra]    Completed announce processing for <604b95adfa625746d1c9e0c18d7cef75>, local rebroadcast limit reached
[2026-04-13 18:11:29] [Extra]    Completed announce processing for <73400f494c8d580bd774443a5163127b>, local rebroadcast limit reached
[2026-04-13 18:11:29] [Extra]    Valid announce for <9e21f49afdaae8885663e807606387a2> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:29] [Extra]    Valid announce for <0f5a6232102792b2e98c19209070d4f8> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:29] [Extra]    Valid announce for <f2406f55c853b6138b3296c2d7763cd8> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:29] [Extra]    Remembering ratchet <8e80ab92dddf07355f90> for <f2406f55c853b6138b3296c2d7763cd8>
[2026-04-13 18:11:29] [Debug]    Destination <f2406f55c853b6138b3296c2d7763cd8> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:30] [Extra]    Valid announce for <f2406f55c853b6138b3296c2d7763cd8> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:30] [Extra]    Valid announce for <893609eb1317367ce3986f04b95973d9> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:30] [Debug]    Destination <893609eb1317367ce3986f04b95973d9> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:30] [Extra]    Valid announce for <f2406f55c853b6138b3296c2d7763cd8> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:30] [Extra]    Valid announce for <893609eb1317367ce3986f04b95973d9> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:30] [Debug]    Rebroadcasting announce for <21d2069f44093a87e42c94c3bd5393b9> with hop count 13
[2026-04-13 18:11:30] [Debug]    Rebroadcasting announce for <794884194914d03c4e199d9c1f090b0c> with hop count 6
[2026-04-13 18:11:30] [Debug]    Rebroadcasting announce for <bb2d98ff84fc6b2ccc7767344343f0f7> with hop count 4
[2026-04-13 18:11:30] [Debug]    Rebroadcasting announce for <7263a18daca33d7629db3ea58818a274> with hop count 7
[2026-04-13 18:11:30] [Debug]    Rebroadcasting announce for <f90b65ba13050cc114363b47066f4e22> with hop count 42
[2026-04-13 18:11:30] [Debug]    Rebroadcasting announce for <9e21f49afdaae8885663e807606387a2> with hop count 4
[2026-04-13 18:11:30] [Debug]    Rebroadcasting announce for <0f5a6232102792b2e98c19209070d4f8> with hop count 4
[2026-04-13 18:11:30] [Debug]    Rebroadcasting announce for <f2406f55c853b6138b3296c2d7763cd8> with hop count 4
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.34ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.03ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.79ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.54ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.35ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.31ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.24ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.21ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.18ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.06ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.99ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.96ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.89ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.85ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.81ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.75ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.68ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.6ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.52ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.49ms
[2026-04-13 18:11:30] [Extra]    Added announce to queue (height 7) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.45ms
[2026-04-13 18:11:30] [Extra]    Valid announce for <893609eb1317367ce3986f04b95973d9> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:30] [Extra]    Heard a rebroadcast of announce for <893609eb1317367ce3986f04b95973d9> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:31] [Extra]    Completed announce processing for <21d2069f44093a87e42c94c3bd5393b9>, local rebroadcast limit reached
[2026-04-13 18:11:31] [Extra]    Completed announce processing for <794884194914d03c4e199d9c1f090b0c>, local rebroadcast limit reached
[2026-04-13 18:11:31] [Extra]    Completed announce processing for <bb2d98ff84fc6b2ccc7767344343f0f7>, local rebroadcast limit reached
[2026-04-13 18:11:31] [Extra]    Completed announce processing for <7263a18daca33d7629db3ea58818a274>, local rebroadcast limit reached
[2026-04-13 18:11:31] [Debug]    Rebroadcasting announce for <3b98ca3ce4b95e607772de2e359cf1b0> with hop count 4
[2026-04-13 18:11:31] [Debug]    Rebroadcasting announce for <eae49b952fed85ac694a6896bda42e4b> with hop count 11
[2026-04-13 18:11:31] [Debug]    Rebroadcasting announce for <893609eb1317367ce3986f04b95973d9> with hop count 4
[2026-04-13 18:11:31] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.58ms
[2026-04-13 18:11:31] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.19ms
[2026-04-13 18:11:31] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.89ms
[2026-04-13 18:11:31] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.6ms
[2026-04-13 18:11:31] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.56ms
[2026-04-13 18:11:31] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.53ms
[2026-04-13 18:11:32] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:32] [Debug]    Replacing destination table entry for <73400f494c8d580bd774443a5163127b> with new announce, since it was more recently emitted
[2026-04-13 18:11:32] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:32] [Extra]    Valid announce for <cf4ca0a1cf91f87778b3543586f75d9f> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:32] [Debug]    Destination <cf4ca0a1cf91f87778b3543586f75d9f> is now 5 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:32] [Extra]    Valid announce for <cd94ae523668995bde8cda5198dd9516> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:32] [Debug]    Destination <cd94ae523668995bde8cda5198dd9516> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:32] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:32] [Extra]    Valid announce for <cf4ca0a1cf91f87778b3543586f75d9f> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:32] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:32] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:32] [Extra]    Valid announce for <cd94ae523668995bde8cda5198dd9516> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:32] [Extra]    Completed announce processing for <3b98ca3ce4b95e607772de2e359cf1b0>, local rebroadcast limit reached
[2026-04-13 18:11:32] [Extra]    Completed announce processing for <eae49b952fed85ac694a6896bda42e4b>, local rebroadcast limit reached
[2026-04-13 18:11:32] [Debug]    Rebroadcasting announce for <0eae1f0c3d08cb59e08fd9e0c6d978d7> with hop count 4
[2026-04-13 18:11:32] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 5
[2026-04-13 18:11:32] [Debug]    Rebroadcasting announce for <73400f494c8d580bd774443a5163127b> with hop count 4
[2026-04-13 18:11:32] [Debug]    Rebroadcasting announce for <cf4ca0a1cf91f87778b3543586f75d9f> with hop count 5
[2026-04-13 18:11:32] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.0ms
[2026-04-13 18:11:32] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.71ms
[2026-04-13 18:11:32] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.5ms
[2026-04-13 18:11:32] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.32ms
[2026-04-13 18:11:32] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.3ms
[2026-04-13 18:11:32] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.29ms
[2026-04-13 18:11:32] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.26ms
[2026-04-13 18:11:32] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.24ms
[2026-04-13 18:11:32] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.22ms
[2026-04-13 18:11:33] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:33] [Extra]    Valid announce for <cf4ca0a1cf91f87778b3543586f75d9f> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:33] [Extra]    Heard a rebroadcast of announce for <cf4ca0a1cf91f87778b3543586f75d9f> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:33] [Extra]    Valid announce for <cd94ae523668995bde8cda5198dd9516> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:33] [Extra]    Heard a rebroadcast of announce for <cd94ae523668995bde8cda5198dd9516> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:33] [Extra]    Valid announce for <5286f603040a5c396ba0dd79d63dc778> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:33] [Extra]    Remembering ratchet <463cf31fc5007c01b8ae> for <5286f603040a5c396ba0dd79d63dc778>
[2026-04-13 18:11:33] [Debug]    Destination <5286f603040a5c396ba0dd79d63dc778> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:11:33] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 18:11:33] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:33] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:11:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:33] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 18:11:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 18:11:33] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:33] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 18:11:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:33] [Extra]    Valid announce for <5286f603040a5c396ba0dd79d63dc778> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:33] [Extra]    Valid announce for <164c25505b1f19d8326fc0d69ca15b4d> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:33] [Debug]    Destination <164c25505b1f19d8326fc0d69ca15b4d> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:33] [Extra]    Valid announce for <4a51b31b70c8461767758f521428cfdd> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:33] [Debug]    Destination <4a51b31b70c8461767758f521428cfdd> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:33] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:33] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:33] [Extra]    Valid announce for <5286f603040a5c396ba0dd79d63dc778> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:33] [Extra]    Valid announce for <164c25505b1f19d8326fc0d69ca15b4d> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:33] [Extra]    Completed announce processing for <0eae1f0c3d08cb59e08fd9e0c6d978d7>, local rebroadcast limit reached
[2026-04-13 18:11:33] [Extra]    Completed announce processing for <2d8a25919ea488ce008d3635d9b104c7>, local rebroadcast limit reached
[2026-04-13 18:11:33] [Debug]    Rebroadcasting announce for <d0d54b0bddc0c4e4f8c6590a8d03149b> with hop count 5
[2026-04-13 18:11:33] [Debug]    Rebroadcasting announce for <ba2780f844f711525924923e9bfb23cb> with hop count 3
[2026-04-13 18:11:33] [Debug]    Rebroadcasting announce for <cd94ae523668995bde8cda5198dd9516> with hop count 4
[2026-04-13 18:11:33] [Debug]    Rebroadcasting announce for <ca273d664d1a6c59a5a002670a641eff> with hop count 3
[2026-04-13 18:11:33] [Debug]    Rebroadcasting announce for <5286f603040a5c396ba0dd79d63dc778> with hop count 5
[2026-04-13 18:11:33] [Debug]    Rebroadcasting announce for <164c25505b1f19d8326fc0d69ca15b4d> with hop count 4
[2026-04-13 18:11:33] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.07ms
[2026-04-13 18:11:33] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.85ms
[2026-04-13 18:11:33] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.75ms
[2026-04-13 18:11:33] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.64ms
[2026-04-13 18:11:33] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.63ms
[2026-04-13 18:11:33] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.61ms
[2026-04-13 18:11:33] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.59ms
[2026-04-13 18:11:33] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.5ms
[2026-04-13 18:11:33] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.48ms
[2026-04-13 18:11:33] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.46ms
[2026-04-13 18:11:33] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.44ms
[2026-04-13 18:11:33] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.43ms
[2026-04-13 18:11:33] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.4ms
[2026-04-13 18:11:33] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.38ms
[2026-04-13 18:11:33] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.37ms
[2026-04-13 18:11:34] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:34] [Extra]    Valid announce for <ca655871b843def1277cc3416cdeed54> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:34] [Debug]    Destination <ca655871b843def1277cc3416cdeed54> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:34] [Extra]    Valid announce for <164c25505b1f19d8326fc0d69ca15b4d> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:34] [Extra]    Heard a rebroadcast of announce for <164c25505b1f19d8326fc0d69ca15b4d> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:34] [Extra]    Valid announce for <4a51b31b70c8461767758f521428cfdd> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:34] [Extra]    Heard a rebroadcast of announce for <4a51b31b70c8461767758f521428cfdd> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:34] [Extra]    Valid announce for <651132825dab903254ab5792691a4be1> 17 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:34] [Extra]    Remembering ratchet <0f5ea23103db3069837c> for <651132825dab903254ab5792691a4be1>
[2026-04-13 18:11:34] [Debug]    Destination <651132825dab903254ab5792691a4be1> is now 17 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:34] [Extra]    Valid announce for <ca655871b843def1277cc3416cdeed54> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:34] [Extra]    Valid announce for <0f57e368cd7ead982478f3640b8c7dc3> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:34] [Debug]    Destination <0f57e368cd7ead982478f3640b8c7dc3> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:34] [Extra]    Valid announce for <4e3bd4191eb994386a841cc3803a2748> 7 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:34] [Debug]    Destination <4e3bd4191eb994386a841cc3803a2748> is now 7 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:34] [Extra]    Valid announce for <4a51b31b70c8461767758f521428cfdd> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:34] [Extra]    Heard a rebroadcast of announce for <4a51b31b70c8461767758f521428cfdd> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:34] [Extra]    Valid announce for <0f57e368cd7ead982478f3640b8c7dc3> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:34] [Extra]    Completed announce processing for <d0d54b0bddc0c4e4f8c6590a8d03149b>, local rebroadcast limit reached
[2026-04-13 18:11:34] [Extra]    Completed announce processing for <ba2780f844f711525924923e9bfb23cb>, local rebroadcast limit reached
[2026-04-13 18:11:34] [Debug]    Rebroadcasting announce for <bab9607fc1b1afbc73bd9b455bf3ee98> with hop count 4
[2026-04-13 18:11:34] [Debug]    Rebroadcasting announce for <110d7f3159c1d306851c3ec5c6d302ef> with hop count 4
[2026-04-13 18:11:34] [Debug]    Rebroadcasting announce for <6ee8d89ae74833c397169c07b81e62e2> with hop count 2
[2026-04-13 18:11:34] [Debug]    Rebroadcasting announce for <3e9008b06eb94497de65274af72143df> with hop count 12
[2026-04-13 18:11:34] [Debug]    Rebroadcasting announce for <4a51b31b70c8461767758f521428cfdd> with hop count 4
[2026-04-13 18:11:34] [Debug]    Rebroadcasting announce for <ca655871b843def1277cc3416cdeed54> with hop count 3
[2026-04-13 18:11:34] [Debug]    Rebroadcasting announce for <651132825dab903254ab5792691a4be1> with hop count 17
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.57ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.2ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.02ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.75ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.69ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.64ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.57ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.54ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.5ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.44ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.41ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.37ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.31ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.28ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.25ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.18ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.99ms
[2026-04-13 18:11:34] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.96ms
[2026-04-13 18:11:35] [Extra]    Valid announce for <0f57e368cd7ead982478f3640b8c7dc3> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:35] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:35] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:35] [Extra]    Completed announce processing for <bab9607fc1b1afbc73bd9b455bf3ee98>, local rebroadcast limit reached
[2026-04-13 18:11:35] [Extra]    Completed announce processing for <6ee8d89ae74833c397169c07b81e62e2>, local rebroadcast limit reached
[2026-04-13 18:11:35] [Extra]    Completed announce processing for <3e9008b06eb94497de65274af72143df>, local rebroadcast limit reached
[2026-04-13 18:11:35] [Debug]    Rebroadcasting announce for <0f57e368cd7ead982478f3640b8c7dc3> with hop count 3
[2026-04-13 18:11:35] [Debug]    Rebroadcasting announce for <4e3bd4191eb994386a841cc3803a2748> with hop count 7
[2026-04-13 18:11:35] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.57ms
[2026-04-13 18:11:35] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.26ms
[2026-04-13 18:11:35] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 7.1ms
[2026-04-13 18:11:36] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:36] [Extra]    Heard a rebroadcast of announce for <2d8a25919ea488ce008d3635d9b104c7> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:36] [Debug]    Path request for <1ba3f953b8e28bd2f5a5ec2e741edf65> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:36] [Debug]    Not answering path request for <1ba3f953b8e28bd2f5a5ec2e741edf65> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242], since next hop is the requestor
[2026-04-13 18:11:36] [Extra]    Valid announce for <f83661aed7c7dd80e1dbbef00dc55ad9> 6 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:36] [Debug]    Destination <f83661aed7c7dd80e1dbbef00dc55ad9> is now 6 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:36] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:36] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:36] [Extra]    Valid announce for <bab9607fc1b1afbc73bd9b455bf3ee98> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:36] [Debug]    Destination <bab9607fc1b1afbc73bd9b455bf3ee98> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:36] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:36] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:36] [Debug]    Rebroadcasting announce for <f90b65ba13050cc114363b47066f4e22> with hop count 42
[2026-04-13 18:11:36] [Debug]    Rebroadcasting announce for <9e21f49afdaae8885663e807606387a2> with hop count 4
[2026-04-13 18:11:36] [Debug]    Rebroadcasting announce for <0f5a6232102792b2e98c19209070d4f8> with hop count 4
[2026-04-13 18:11:36] [Debug]    Rebroadcasting announce for <f2406f55c853b6138b3296c2d7763cd8> with hop count 4
[2026-04-13 18:11:36] [Debug]    Rebroadcasting announce for <ca273d664d1a6c59a5a002670a641eff> with hop count 3
[2026-04-13 18:11:36] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 4
[2026-04-13 18:11:36] [Debug]    Rebroadcasting announce for <f83661aed7c7dd80e1dbbef00dc55ad9> with hop count 6
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.99ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.75ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.66ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.49ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.47ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.46ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.43ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.42ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.38ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.29ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.26ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.25ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.22ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.21ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.18ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.08ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.05ms
[2026-04-13 18:11:36] [Extra]    Added announce to queue (height 6) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.04ms
[2026-04-13 18:11:37] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:37] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:37] [Extra]    Valid announce for <f83661aed7c7dd80e1dbbef00dc55ad9> 7 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:37] [Extra]    Heard a rebroadcast of announce for <f83661aed7c7dd80e1dbbef00dc55ad9> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:37] [Extra]    Valid announce for <bab9607fc1b1afbc73bd9b455bf3ee98> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:37] [Extra]    Heard a rebroadcast of announce for <bab9607fc1b1afbc73bd9b455bf3ee98> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:37] [Extra]    Valid announce for <c1c04c5dfdb2d647a0a5e6bf54bc4b07> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:37] [Extra]    Remembering ratchet <3c2e99daf70089e2242a> for <c1c04c5dfdb2d647a0a5e6bf54bc4b07>
[2026-04-13 18:11:37] [Debug]    Destination <c1c04c5dfdb2d647a0a5e6bf54bc4b07> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:37] [Extra]    Completed announce processing for <f90b65ba13050cc114363b47066f4e22>, local rebroadcast limit reached
[2026-04-13 18:11:37] [Extra]    Completed announce processing for <9e21f49afdaae8885663e807606387a2>, local rebroadcast limit reached
[2026-04-13 18:11:37] [Extra]    Completed announce processing for <0f5a6232102792b2e98c19209070d4f8>, local rebroadcast limit reached
[2026-04-13 18:11:37] [Extra]    Completed announce processing for <f2406f55c853b6138b3296c2d7763cd8>, local rebroadcast limit reached
[2026-04-13 18:11:37] [Debug]    Rebroadcasting announce for <893609eb1317367ce3986f04b95973d9> with hop count 4
[2026-04-13 18:11:37] [Debug]    Rebroadcasting announce for <73400f494c8d580bd774443a5163127b> with hop count 3
[2026-04-13 18:11:37] [Debug]    Rebroadcasting announce for <bab9607fc1b1afbc73bd9b455bf3ee98> with hop count 4
[2026-04-13 18:11:37] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.96ms
[2026-04-13 18:11:37] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.58ms
[2026-04-13 18:11:37] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.41ms
[2026-04-13 18:11:37] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.17ms
[2026-04-13 18:11:37] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.15ms
[2026-04-13 18:11:37] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.13ms
[2026-04-13 18:11:37] [Extra]    Valid announce for <42f9397c4362faa9cf62ac6da6a41f5a> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:37] [Debug]    Destination <42f9397c4362faa9cf62ac6da6a41f5a> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:38] [Extra]    Valid announce for <c1c04c5dfdb2d647a0a5e6bf54bc4b07> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:38] [Extra]    Valid announce for <8f396811d9d704a0237e09103ddec1eb> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:38] [Debug]    Destination <8f396811d9d704a0237e09103ddec1eb> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:38] [Extra]    Valid announce for <42f9397c4362faa9cf62ac6da6a41f5a> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:38] [Extra]    Heard a rebroadcast of announce for <42f9397c4362faa9cf62ac6da6a41f5a> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:11:38] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:38] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:11:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 18:11:38] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:38] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 18:11:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 18:11:38] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:38] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 18:11:38] [Extra]    Valid announce for <8f396811d9d704a0237e09103ddec1eb> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:38] [Extra]    Valid announce for <9f233c2f8499e84df0a58c63ac2ae728> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:38] [Extra]    Remembering ratchet <ed8fb21d75938b2e90f0> for <9f233c2f8499e84df0a58c63ac2ae728>
[2026-04-13 18:11:38] [Debug]    Destination <9f233c2f8499e84df0a58c63ac2ae728> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:38] [Extra]    Completed announce processing for <893609eb1317367ce3986f04b95973d9>, local rebroadcast limit reached
[2026-04-13 18:11:38] [Debug]    Rebroadcasting announce for <cf4ca0a1cf91f87778b3543586f75d9f> with hop count 5
[2026-04-13 18:11:38] [Debug]    Rebroadcasting announce for <c1c04c5dfdb2d647a0a5e6bf54bc4b07> with hop count 4
[2026-04-13 18:11:38] [Debug]    Rebroadcasting announce for <42f9397c4362faa9cf62ac6da6a41f5a> with hop count 4
[2026-04-13 18:11:38] [Debug]    Rebroadcasting announce for <8f396811d9d704a0237e09103ddec1eb> with hop count 5
[2026-04-13 18:11:38] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.0ms
[2026-04-13 18:11:38] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.16ms
[2026-04-13 18:11:38] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.39ms
[2026-04-13 18:11:38] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.93ms
[2026-04-13 18:11:38] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.9ms
[2026-04-13 18:11:38] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.86ms
[2026-04-13 18:11:38] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.82ms
[2026-04-13 18:11:38] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.8ms
[2026-04-13 18:11:38] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.77ms
[2026-04-13 18:11:38] [Extra]    Valid announce for <9f233c2f8499e84df0a58c63ac2ae728> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:38] [Extra]    Valid announce for <23dfe3ab2b2523179a9fe1f22c18c13e> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:38] [Debug]    Destination <23dfe3ab2b2523179a9fe1f22c18c13e> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:38] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:38] [Debug]    Destination <e80bd281bd26e00d735aada7b7b94c7a> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:39] [Extra]    Valid announce for <9f233c2f8499e84df0a58c63ac2ae728> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:39] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:39] [Extra]    Heard a rebroadcast of announce for <e80bd281bd26e00d735aada7b7b94c7a> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:39] [Extra]    Completed announce processing for <cf4ca0a1cf91f87778b3543586f75d9f>, local rebroadcast limit reached
[2026-04-13 18:11:39] [Debug]    Rebroadcasting announce for <cd94ae523668995bde8cda5198dd9516> with hop count 4
[2026-04-13 18:11:39] [Debug]    Rebroadcasting announce for <5286f603040a5c396ba0dd79d63dc778> with hop count 5
[2026-04-13 18:11:39] [Debug]    Rebroadcasting announce for <164c25505b1f19d8326fc0d69ca15b4d> with hop count 4
[2026-04-13 18:11:39] [Debug]    Rebroadcasting announce for <9f233c2f8499e84df0a58c63ac2ae728> with hop count 3
[2026-04-13 18:11:39] [Debug]    Rebroadcasting announce for <23dfe3ab2b2523179a9fe1f22c18c13e> with hop count 4
[2026-04-13 18:11:39] [Debug]    Rebroadcasting announce for <e80bd281bd26e00d735aada7b7b94c7a> with hop count 4
[2026-04-13 18:11:39] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.91ms
[2026-04-13 18:11:39] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.6ms
[2026-04-13 18:11:39] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.38ms
[2026-04-13 18:11:39] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.19ms
[2026-04-13 18:11:39] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.15ms
[2026-04-13 18:11:39] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.12ms
[2026-04-13 18:11:39] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.05ms
[2026-04-13 18:11:39] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.02ms
[2026-04-13 18:11:39] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.99ms
[2026-04-13 18:11:39] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.93ms
[2026-04-13 18:11:39] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.9ms
[2026-04-13 18:11:39] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.86ms
[2026-04-13 18:11:39] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 5.8ms
[2026-04-13 18:11:39] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 5.77ms
[2026-04-13 18:11:39] [Extra]    Added announce to queue (height 5) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 5.74ms
[2026-04-13 18:11:40] [Extra]    Valid announce for <23dfe3ab2b2523179a9fe1f22c18c13e> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:40] [Extra]    Heard a rebroadcast of announce for <23dfe3ab2b2523179a9fe1f22c18c13e> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:40] [Debug]    Rebroadcasting announce for <110d7f3159c1d306851c3ec5c6d302ef> with hop count 4
[2026-04-13 18:11:40] [Extra]    Completed announce processing for <cd94ae523668995bde8cda5198dd9516>, local rebroadcast limit reached
[2026-04-13 18:11:40] [Extra]    Completed announce processing for <5286f603040a5c396ba0dd79d63dc778>, local rebroadcast limit reached
[2026-04-13 18:11:40] [Extra]    Completed announce processing for <164c25505b1f19d8326fc0d69ca15b4d>, local rebroadcast limit reached
[2026-04-13 18:11:40] [Debug]    Rebroadcasting announce for <4a51b31b70c8461767758f521428cfdd> with hop count 4
[2026-04-13 18:11:40] [Debug]    Rebroadcasting announce for <ca655871b843def1277cc3416cdeed54> with hop count 3
[2026-04-13 18:11:40] [Debug]    Rebroadcasting announce for <651132825dab903254ab5792691a4be1> with hop count 17
[2026-04-13 18:11:40] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.39ms
[2026-04-13 18:11:40] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.12ms
[2026-04-13 18:11:40] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.85ms
[2026-04-13 18:11:40] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.58ms
[2026-04-13 18:11:40] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.54ms
[2026-04-13 18:11:40] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.5ms
[2026-04-13 18:11:40] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.43ms
[2026-04-13 18:11:40] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.31ms
[2026-04-13 18:11:40] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.27ms
[2026-04-13 18:11:40] [Extra]    Valid announce for <aea9a5706da57d932ffa95cf58d0bac8> 14 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:40] [Extra]    Valid announce for <55c16056b1b0f93042c92ea31ceb801a> 20 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:40] [Extra]    Valid announce for <284395a82e44594554a03e3757988396> 24 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:41] [Extra]    Valid announce for <cf4ca0a1cf91f87778b3543586f75d9f> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:41] [Debug]    Destination <cf4ca0a1cf91f87778b3543586f75d9f> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:41] [Extra]    Completed announce processing for <110d7f3159c1d306851c3ec5c6d302ef>, local rebroadcast limit reached
[2026-04-13 18:11:41] [Extra]    Completed announce processing for <4a51b31b70c8461767758f521428cfdd>, local rebroadcast limit reached
[2026-04-13 18:11:41] [Extra]    Completed announce processing for <ca655871b843def1277cc3416cdeed54>, local rebroadcast limit reached
[2026-04-13 18:11:41] [Extra]    Completed announce processing for <651132825dab903254ab5792691a4be1>, local rebroadcast limit reached
[2026-04-13 18:11:41] [Debug]    Rebroadcasting announce for <0f57e368cd7ead982478f3640b8c7dc3> with hop count 3
[2026-04-13 18:11:41] [Debug]    Rebroadcasting announce for <4e3bd4191eb994386a841cc3803a2748> with hop count 7
[2026-04-13 18:11:41] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 7.58ms
[2026-04-13 18:11:41] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 7.18ms
[2026-04-13 18:11:41] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.91ms
[2026-04-13 18:11:41] [Extra]    Valid announce for <cf4ca0a1cf91f87778b3543586f75d9f> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:42] [Extra]    Valid announce for <cf4ca0a1cf91f87778b3543586f75d9f> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:42] [Extra]    Heard a rebroadcast of announce for <cf4ca0a1cf91f87778b3543586f75d9f> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:42] [Extra]    Valid announce for <cd94ae523668995bde8cda5198dd9516> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:42] [Debug]    Destination <cd94ae523668995bde8cda5198dd9516> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:42] [Debug]    Rebroadcasting announce for <ca273d664d1a6c59a5a002670a641eff> with hop count 3
[2026-04-13 18:11:42] [Extra]    Completed announce processing for <0f57e368cd7ead982478f3640b8c7dc3>, local rebroadcast limit reached
[2026-04-13 18:11:42] [Extra]    Completed announce processing for <4e3bd4191eb994386a841cc3803a2748>, local rebroadcast limit reached
[2026-04-13 18:11:42] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 4
[2026-04-13 18:11:42] [Debug]    Rebroadcasting announce for <f83661aed7c7dd80e1dbbef00dc55ad9> with hop count 6
[2026-04-13 18:11:42] [Debug]    Rebroadcasting announce for <cf4ca0a1cf91f87778b3543586f75d9f> with hop count 4
[2026-04-13 18:11:42] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 11.46ms
[2026-04-13 18:11:42] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 11.27ms
[2026-04-13 18:11:42] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 11.16ms
[2026-04-13 18:11:42] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 11.07ms
[2026-04-13 18:11:42] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 11.06ms
[2026-04-13 18:11:42] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 11.05ms
[2026-04-13 18:11:42] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 11.02ms
[2026-04-13 18:11:42] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 11.01ms
[2026-04-13 18:11:42] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 10.99ms
[2026-04-13 18:11:42] [Extra]    Valid announce for <cd94ae523668995bde8cda5198dd9516> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:43] [Extra]    Valid announce for <cd94ae523668995bde8cda5198dd9516> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:43] [Extra]    Heard a rebroadcast of announce for <cd94ae523668995bde8cda5198dd9516> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 18:11:43] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:43] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 18:11:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:11:43] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:43] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:11:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 18:11:43] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:43] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 18:11:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:43] [Extra]    Valid announce for <3b0e16f84e64170294aadab9d360bac3> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:43] [Debug]    Destination <3b0e16f84e64170294aadab9d360bac3> is now 5 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:43] [Debug]    Rebroadcasting announce for <73400f494c8d580bd774443a5163127b> with hop count 3
[2026-04-13 18:11:43] [Extra]    Completed announce processing for <ca273d664d1a6c59a5a002670a641eff>, local rebroadcast limit reached
[2026-04-13 18:11:43] [Extra]    Completed announce processing for <2d8a25919ea488ce008d3635d9b104c7>, local rebroadcast limit reached
[2026-04-13 18:11:43] [Extra]    Completed announce processing for <f83661aed7c7dd80e1dbbef00dc55ad9>, local rebroadcast limit reached
[2026-04-13 18:11:43] [Debug]    Rebroadcasting announce for <bab9607fc1b1afbc73bd9b455bf3ee98> with hop count 4
[2026-04-13 18:11:43] [Debug]    Rebroadcasting announce for <cd94ae523668995bde8cda5198dd9516> with hop count 4
[2026-04-13 18:11:43] [Debug]    Rebroadcasting announce for <3b0e16f84e64170294aadab9d360bac3> with hop count 5
[2026-04-13 18:11:43] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.95ms
[2026-04-13 18:11:43] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.66ms
[2026-04-13 18:11:43] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.43ms
[2026-04-13 18:11:43] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.34ms
[2026-04-13 18:11:43] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.32ms
[2026-04-13 18:11:43] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.31ms
[2026-04-13 18:11:43] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 6.29ms
[2026-04-13 18:11:43] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 6.27ms
[2026-04-13 18:11:43] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 6.26ms
[2026-04-13 18:11:44] [Extra]    Valid announce for <3b0e16f84e64170294aadab9d360bac3> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:44] [Extra]    Valid announce for <3b0e16f84e64170294aadab9d360bac3> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:44] [Extra]    Heard a rebroadcast of announce for <3b0e16f84e64170294aadab9d360bac3> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:44] [Extra]    Valid announce for <249922b43687681be4d0a025507ef1ae> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:44] [Debug]    Destination <249922b43687681be4d0a025507ef1ae> is now 5 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:44] [Extra]    Completed announce processing for <73400f494c8d580bd774443a5163127b>, local rebroadcast limit reached
[2026-04-13 18:11:44] [Extra]    Completed announce processing for <bab9607fc1b1afbc73bd9b455bf3ee98>, local rebroadcast limit reached
[2026-04-13 18:11:44] [Debug]    Rebroadcasting announce for <c1c04c5dfdb2d647a0a5e6bf54bc4b07> with hop count 4
[2026-04-13 18:11:44] [Debug]    Rebroadcasting announce for <42f9397c4362faa9cf62ac6da6a41f5a> with hop count 4
[2026-04-13 18:11:44] [Debug]    Rebroadcasting announce for <8f396811d9d704a0237e09103ddec1eb> with hop count 5
[2026-04-13 18:11:44] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 9.81ms
[2026-04-13 18:11:44] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 9.61ms
[2026-04-13 18:11:44] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 9.49ms
[2026-04-13 18:11:44] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 9.37ms
[2026-04-13 18:11:44] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 9.34ms
[2026-04-13 18:11:44] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 9.32ms
[2026-04-13 18:11:45] [Extra]    Valid announce for <249922b43687681be4d0a025507ef1ae> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:45] [Extra]    Heard a rebroadcast of announce for <249922b43687681be4d0a025507ef1ae> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:45] [Extra]    Valid announce for <249922b43687681be4d0a025507ef1ae> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:45] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:45] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:45] [Debug]    Path request for <54a44a780e15627fa6bb8e693a5c5d09> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:45] [Debug]    Ignoring path request for <54a44a780e15627fa6bb8e693a5c5d09> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 18:11:45] [Extra]    Completed announce processing for <c1c04c5dfdb2d647a0a5e6bf54bc4b07>, local rebroadcast limit reached
[2026-04-13 18:11:45] [Extra]    Completed announce processing for <42f9397c4362faa9cf62ac6da6a41f5a>, local rebroadcast limit reached
[2026-04-13 18:11:45] [Extra]    Completed announce processing for <8f396811d9d704a0237e09103ddec1eb>, local rebroadcast limit reached
[2026-04-13 18:11:45] [Debug]    Rebroadcasting announce for <9f233c2f8499e84df0a58c63ac2ae728> with hop count 3
[2026-04-13 18:11:45] [Debug]    Rebroadcasting announce for <23dfe3ab2b2523179a9fe1f22c18c13e> with hop count 4
[2026-04-13 18:11:45] [Debug]    Rebroadcasting announce for <e80bd281bd26e00d735aada7b7b94c7a> with hop count 4
[2026-04-13 18:11:45] [Debug]    Rebroadcasting announce for <249922b43687681be4d0a025507ef1ae> with hop count 5
[2026-04-13 18:11:45] [Debug]    Rebroadcasting announce for <2d8a25919ea488ce008d3635d9b104c7> with hop count 4
[2026-04-13 18:11:45] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 9.29ms
[2026-04-13 18:11:45] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 9.05ms
[2026-04-13 18:11:45] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.92ms
[2026-04-13 18:11:45] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.77ms
[2026-04-13 18:11:45] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.65ms
[2026-04-13 18:11:45] [Extra]    Added announce to queue (height 2) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.63ms
[2026-04-13 18:11:45] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.59ms
[2026-04-13 18:11:45] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.58ms
[2026-04-13 18:11:45] [Extra]    Added announce to queue (height 3) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.56ms
[2026-04-13 18:11:45] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.53ms
[2026-04-13 18:11:45] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.51ms
[2026-04-13 18:11:45] [Extra]    Added announce to queue (height 4) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.49ms
[2026-04-13 18:11:46] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:46] [Extra]    Heard a rebroadcast of announce for <2d8a25919ea488ce008d3635d9b104c7> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:46] [Extra]    Valid announce for <125bfe4fc66ac35f73e30c2613ab09fa> 7 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:46] [Extra]    Valid announce for <c79149286d07d2b2be3f167405eedda6> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:46] [Extra]    Valid announce for <c266fa77af262ed1cde8bc66b2f716a3> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:46] [Debug]    Destination <c266fa77af262ed1cde8bc66b2f716a3> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 18:11:46] [Debug]    Path request for <4447b200737e3fead7d054e5fc58e081> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:46] [Debug]    Answering path request for <4447b200737e3fead7d054e5fc58e081> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], path is known
[2026-04-13 18:11:46] [Extra]    Completed announce processing for <9f233c2f8499e84df0a58c63ac2ae728>, local rebroadcast limit reached
[2026-04-13 18:11:46] [Extra]    Completed announce processing for <23dfe3ab2b2523179a9fe1f22c18c13e>, local rebroadcast limit reached
[2026-04-13 18:11:46] [Extra]    Completed announce processing for <e80bd281bd26e00d735aada7b7b94c7a>, local rebroadcast limit reached
[2026-04-13 18:11:46] [Debug]    Rebroadcasting announce for <c266fa77af262ed1cde8bc66b2f716a3> with hop count 3
[2026-04-13 18:11:46] [Extra]    Valid announce for <dd9a64e30bf202e6ad95bc14666ae358> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:46] [Extra]    Remembering ratchet <bac19280b4aede4b3671> for <dd9a64e30bf202e6ad95bc14666ae358>
[2026-04-13 18:11:46] [Debug]    Destination <dd9a64e30bf202e6ad95bc14666ae358> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:47] [Extra]    Valid announce for <c266fa77af262ed1cde8bc66b2f716a3> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:47] [Extra]    Rebroadcasted announce for <c266fa77af262ed1cde8bc66b2f716a3> has been passed on to another node, no further tries needed
[2026-04-13 18:11:47] [Extra]    Valid announce for <dd9a64e30bf202e6ad95bc14666ae358> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:47] [Extra]    Heard a rebroadcast of announce for <dd9a64e30bf202e6ad95bc14666ae358> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:47] [Extra]    Valid announce for <b70dbc0b123dfc531164dd8189d6ac95> 8 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:47] [Debug]    Destination <b70dbc0b123dfc531164dd8189d6ac95> is now 8 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 18:11:47] [Debug]    Rebroadcasting announce as path response for <4447b200737e3fead7d054e5fc58e081> with hop count 4
[2026-04-13 18:11:47] [Debug]    Rebroadcasting announce for <dd9a64e30bf202e6ad95bc14666ae358> with hop count 4
[2026-04-13 18:11:47] [Debug]    Rebroadcasting announce for <b70dbc0b123dfc531164dd8189d6ac95> with hop count 8
[2026-04-13 18:11:47] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] for processing in 8.75ms
[2026-04-13 18:11:47] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] for processing in 8.51ms
[2026-04-13 18:11:47] [Extra]    Added announce to queue (height 1) on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] for processing in 8.38ms
[2026-04-13 18:11:47] [Extra]    Valid announce for <4b706e311fd6961031885677cf01153a> 15 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:47] [Extra]    Remembering ratchet <e5e439a2d2c47cadfcd3> for <4b706e311fd6961031885677cf01153a>
[2026-04-13 18:11:47] [Debug]    Destination <4b706e311fd6961031885677cf01153a> is now 15 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 18:11:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 18:11:48] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:48] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 18:11:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 18:11:48] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 18:11:48] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 18:11:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 18:11:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 18:11:48] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
