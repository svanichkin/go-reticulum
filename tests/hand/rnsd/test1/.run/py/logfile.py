Exception in thread Thread-4 (reconnect):
Traceback (most recent call last):
  File "/Library/Frameworks/Python.framework/Versions/3.10/lib/python3.10/threading.py", line 1016, in _bootstrap_inner
    self.run()
  File "/Library/Frameworks/Python.framework/Versions/3.10/lib/python3.10/threading.py", line 953, in run
    self._target(*self._args, **self._kwargs)
  File "/Users/alien/Library/Python/3.10/lib/python/site-packages/RNS/Interfaces/TCPInterface.py", line 294, in reconnect
    RNS.Transport.synthesize_tunnel(self)
  File "/Users/alien/Library/Python/3.10/lib/python/site-packages/RNS/Transport.py", line 2027, in synthesize_tunnel
    public_key     = RNS.Transport.identity.get_public_key()
AttributeError: 'NoneType' object has no attribute 'get_public_key'
Exception in thread Thread-7 (reconnect):
Traceback (most recent call last):
  File "/Library/Frameworks/Python.framework/Versions/3.10/lib/python3.10/threading.py", line 1016, in _bootstrap_inner
    self.run()
  File "/Library/Frameworks/Python.framework/Versions/3.10/lib/python3.10/threading.py", line 953, in run
    self._target(*self._args, **self._kwargs)
  File "/Users/alien/Library/Python/3.10/lib/python/site-packages/RNS/Interfaces/TCPInterface.py", line 294, in reconnect
    RNS.Transport.synthesize_tunnel(self)
  File "/Users/alien/Library/Python/3.10/lib/python/site-packages/RNS/Transport.py", line 2027, in synthesize_tunnel
    public_key     = RNS.Transport.identity.get_public_key()
AttributeError: 'NoneType' object has no attribute 'get_public_key'
[2026-04-13 17:40:28] [Debug]    Reticulum running in interpreted mode
[2026-04-13 17:40:28] [Debug]    Started shared instance interface: Shared Instance[37430]
[2026-04-13 17:40:28] [Debug]    Cleaning ratchets...
[2026-04-13 17:40:28] [Extra]    Cleaning resource and packet caches...
[2026-04-13 17:40:28] [Verbose]  Bringing up system interfaces...
[2026-04-13 17:40:28] [Debug]    Establishing TCP connection for TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]...
[2026-04-13 17:40:28] [Debug]    TCP connection for TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] established
[2026-04-13 17:40:28] [Debug]    TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] hardware MTU set to 8192
[2026-04-13 17:40:28] [Debug]    Establishing TCP connection for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]...
[2026-04-13 17:40:28] [Error]    Initial connection for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] could not be established: [Errno 8] nodename nor servname provided, or not known
[2026-04-13 17:40:28] [Error]    Leaving unconnected and retrying connection in 5 seconds.
[2026-04-13 17:40:28] [Debug]    TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] hardware MTU set to 8192
[2026-04-13 17:40:28] [Debug]    Establishing TCP connection for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]...
[2026-04-13 17:40:28] [Debug]    TCP connection for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] established
[2026-04-13 17:40:28] [Debug]    TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] hardware MTU set to 8192
[2026-04-13 17:40:28] [Debug]    Establishing TCP connection for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]...
[2026-04-13 17:40:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:40:33] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:40:33] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:40:33] [Error]    Initial connection for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] could not be established: timed out
[2026-04-13 17:40:33] [Error]    Leaving unconnected and retrying connection in 5 seconds.
[2026-04-13 17:40:33] [Debug]    TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] hardware MTU set to 8192
[2026-04-13 17:40:33] [Debug]    Establishing TCP connection for TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]...
[2026-04-13 17:40:33] [Debug]    TCP connection for TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] established
[2026-04-13 17:40:33] [Debug]    TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] hardware MTU set to 8192
[2026-04-13 17:40:33] [Debug]    Establishing TCP connection for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]...
[2026-04-13 17:40:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:40:38] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:40:38] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:40:38] [Warning]  An interface error occurred for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965], the contained exception was: 'NoneType' object has no attribute 'get_public_key'
[2026-04-13 17:40:38] [Warning]  Attempting to reconnect...
[2026-04-13 17:40:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:40:38] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:40:38] [Error]    Initial connection for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] could not be established: timed out
[2026-04-13 17:40:38] [Error]    Leaving unconnected and retrying connection in 5 seconds.
[2026-04-13 17:40:38] [Debug]    TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] hardware MTU set to 8192
[2026-04-13 17:40:38] [Verbose]  System interfaces are ready
[2026-04-13 17:40:38] [Debug]    Utilising cryptography backend "openssl, PyCA 46.0.3"
[2026-04-13 17:40:38] [Verbose]  Configuration loaded from /Users/alien/Vault/Projects/Self/Golang/Reticulum/go-reticulum/tests/hand/rnsd/test1/.run/py/config
[2026-04-13 17:40:38] [Verbose]  Loaded 1839 known destination from storage
[2026-04-13 17:40:38] [Verbose]  Loaded Transport Identity from storage
[2026-04-13 17:40:38] [Notice]   Started rnsd version 1.0.4
[2026-04-13 17:40:38] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:38] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:39] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:40:39] [Debug]    Destination <e80bd281bd26e00d735aada7b7b94c7a> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:40:39] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:40] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:40] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:41] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:41] [Debug]    Destination <29b2ebe588859e48aabf13e97cfe245b> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:42] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:40:42] [Extra]    Valid announce for <26942c55352995532ae4965b0343db7f> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:40:42] [Debug]    Destination <26942c55352995532ae4965b0343db7f> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:40:42] [Extra]    Valid announce for <26942c55352995532ae4965b0343db7f> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:40:43] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:40:43] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:40:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:40:43] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-13 17:40:43] [Warning]  Attempting to reconnect...
[2026-04-13 17:40:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:40:43] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:40:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:40:43] [Extra]    Valid announce for <75a761de7bdd03adeb6ad16b004e53a1> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:43] [Debug]    Destination <75a761de7bdd03adeb6ad16b004e53a1> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:48] [Extra]    Valid announce for <103eb3c7f35278ba33e7d014e341b3ec> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:48] [Debug]    Destination <103eb3c7f35278ba33e7d014e341b3ec> is now 6 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:40:48] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:40:48] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:40:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:40:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:40:48] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:40:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:40:48] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-13 17:40:48] [Warning]  Attempting to reconnect...
[2026-04-13 17:40:49] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:40:49] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:40:50] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:50] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:50] [Debug]    Destination <e345f6220682e127cab52c3387436778> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:50] [Debug]    Path request for <136f555ecf56ce8c46f300b39046de94> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:40:50] [Debug]    Ignoring path request for <136f555ecf56ce8c46f300b39046de94> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:40:51] [Extra]    Valid announce for <9eb0f17e1691e819bb34eddafa2ca82b> 7 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:51] [Debug]    Destination <9eb0f17e1691e819bb34eddafa2ca82b> is now 7 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:51] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 8 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:51] [Debug]    Destination <02aaf088472435718061211d3752c8ed> is now 8 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:51] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:51] [Extra]    Remembering ratchet <7b7b21123927ea513839> for <794884194914d03c4e199d9c1f090b0c>
[2026-04-13 17:40:51] [Debug]    Destination <794884194914d03c4e199d9c1f090b0c> is now 6 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:52] [Extra]    Valid announce for <54f0e7796ac804890832cb3ee61131f2> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:52] [Extra]    Remembering ratchet <79846146397d6ee40d69> for <54f0e7796ac804890832cb3ee61131f2>
[2026-04-13 17:40:52] [Debug]    Destination <54f0e7796ac804890832cb3ee61131f2> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:52] [Extra]    Valid announce for <dce1135780d75f28804a92545aea418a> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:52] [Debug]    Destination <dce1135780d75f28804a92545aea418a> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:40:53] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:40:53] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:40:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:40:53] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-13 17:40:53] [Warning]  Attempting to reconnect...
[2026-04-13 17:40:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:40:53] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:40:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:40:56] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:40:56] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:40:56] [Debug]    Path request for <5de01254e806ab49d9c348ef2da1b7ad> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:56] [Debug]    Ignoring path request for <5de01254e806ab49d9c348ef2da1b7ad> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242], no path known
[2026-04-13 17:40:57] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:40:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:40:58] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:40:58] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:40:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:40:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:40:58] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:40:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:40:58] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-13 17:40:58] [Warning]  Attempting to reconnect...
[2026-04-13 17:41:00] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:00] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:00] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:03] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:03] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:41:03] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:03] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:41:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:03] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-13 17:41:03] [Warning]  Attempting to reconnect...
[2026-04-13 17:41:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:41:03] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:04] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:05] [Debug]    Path request for <52ad03c20beb735c46a69a88da290717> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:05] [Debug]    Ignoring path request for <52ad03c20beb735c46a69a88da290717> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242], no path known
[2026-04-13 17:41:06] [Debug]    Path request for <995cc3851347f138f51af73d182d5e1d> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:41:06] [Debug]    Ignoring path request for <995cc3851347f138f51af73d182d5e1d> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:41:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:41:08] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:08] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:41:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:41:08] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:08] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-13 17:41:08] [Warning]  Attempting to reconnect...
[2026-04-13 17:41:10] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:10] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:10] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:11] [Debug]    Path request for <091ec10a646b0c00cd246f088f7fa907> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:11] [Debug]    Ignoring path request for <091ec10a646b0c00cd246f088f7fa907> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242], no path known
[2026-04-13 17:41:11] [Debug]    Path request for <41165bf22801b29880cdf766ed8cceaf> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:41:11] [Debug]    Ignoring path request for <41165bf22801b29880cdf766ed8cceaf> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:41:12] [Debug]    Path request for <3b171e0b79acf468ae1bf3a6d8515d12> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:12] [Debug]    Ignoring path request for <3b171e0b79acf468ae1bf3a6d8515d12> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242], no path known
[2026-04-13 17:41:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:41:13] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:13] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:41:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:13] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-13 17:41:13] [Warning]  Attempting to reconnect...
[2026-04-13 17:41:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:41:13] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:13] [Debug]    Path request for <a3dfea289534e4b2d0f5730310eebd99> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:13] [Debug]    Ignoring path request for <a3dfea289534e4b2d0f5730310eebd99> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242], no path known
[2026-04-13 17:41:13] [Debug]    Ignoring duplicate path request for <a3dfea289534e4b2d0f5730310eebd99> with tag <a3dfea289534e4b2d0f5730310eebd9947238abb0c1a23d040ab79904773d62f>
[2026-04-13 17:41:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:41:18] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:18] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:41:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:41:18] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:18] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-13 17:41:18] [Warning]  Attempting to reconnect...
[2026-04-13 17:41:19] [Debug]    Path request for <177ab81a7ff8259290bd886b64360ead> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:19] [Debug]    Ignoring path request for <177ab81a7ff8259290bd886b64360ead> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242], no path known
[2026-04-13 17:41:19] [Debug]    Ignoring duplicate path request for <177ab81a7ff8259290bd886b64360ead> with tag <177ab81a7ff8259290bd886b64360ead395fdb28cea1118beb072fae49feff00>
[2026-04-13 17:41:20] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:20] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:20] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:20] [Debug]    Destination <02aaf088472435718061211d3752c8ed> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:20] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:20] [Debug]    Destination <e345f6220682e127cab52c3387436778> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:21] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:21] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:21] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:41:23] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:23] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:41:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:23] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-13 17:41:23] [Warning]  Attempting to reconnect...
[2026-04-13 17:41:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:41:23] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:23] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 7 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:23] [Debug]    Replacing destination table entry for <794884194914d03c4e199d9c1f090b0c> with new announce, since it was more recently emitted
[2026-04-13 17:41:23] [Debug]    Destination <794884194914d03c4e199d9c1f090b0c> is now 7 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:24] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 9 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:27] [Debug]    Path request for <053db2ffc4105a601f6ac5f23cc356d6> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:41:27] [Debug]    Ignoring path request for <053db2ffc4105a601f6ac5f23cc356d6> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:41:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:41:28] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:28] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:41:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:41:28] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:28] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-13 17:41:28] [Warning]  Attempting to reconnect...
[2026-04-13 17:41:28] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:28] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:29] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:31] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:31] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:31] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:41:33] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:33] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:41:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:33] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-13 17:41:33] [Warning]  Attempting to reconnect...
[2026-04-13 17:41:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:41:33] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:41:38] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:38] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:41:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:41:38] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:38] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-13 17:41:38] [Warning]  Attempting to reconnect...
[2026-04-13 17:41:39] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:39] [Debug]    Destination <e80bd281bd26e00d735aada7b7b94c7a> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:40] [Debug]    Path request for <e1a8a3a6644ef291bf0543bac268498d> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:40] [Debug]    Ignoring path request for <e1a8a3a6644ef291bf0543bac268498d> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242], no path known
[2026-04-13 17:41:40] [Debug]    Path request for <e1a8a3a6644ef291bf0543bac268498d> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:41:40] [Debug]    Ignoring path request for <e1a8a3a6644ef291bf0543bac268498d> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:41:40] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:41] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:41] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:41] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:41] [Extra]    Valid announce for <dc1665cfd1f79fb83b430d953bb13f59> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:41] [Extra]    Remembering ratchet <5db08ab5e2251a2b56aa> for <dc1665cfd1f79fb83b430d953bb13f59>
[2026-04-13 17:41:41] [Debug]    Destination <dc1665cfd1f79fb83b430d953bb13f59> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:41] [Extra]    Valid announce for <dc1665cfd1f79fb83b430d953bb13f59> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:41:42] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:42] [Debug]    Destination <29b2ebe588859e48aabf13e97cfe245b> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:42] [Extra]    Valid announce for <33f4b51ed94310425808f2e84ffb918c> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:42] [Extra]    Remembering ratchet <dba455d958a5f4df967d> for <33f4b51ed94310425808f2e84ffb918c>
[2026-04-13 17:41:42] [Debug]    Destination <33f4b51ed94310425808f2e84ffb918c> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:42] [Extra]    Valid announce for <ea4a5a9a01c9ad6e39718716e6cf9c06> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:42] [Extra]    Remembering ratchet <4f2478c9901fa8b7fded> for <ea4a5a9a01c9ad6e39718716e6cf9c06>
[2026-04-13 17:41:42] [Debug]    Destination <ea4a5a9a01c9ad6e39718716e6cf9c06> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:42] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:41:43] [Extra]    Valid announce for <ea4a5a9a01c9ad6e39718716e6cf9c06> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:43] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:43] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: [Errno 60] Operation timed out
[2026-04-13 17:41:43] [Warning]  Attempting to reconnect...
[2026-04-13 17:41:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:41:43] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:43] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:41:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:41:43] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:41:48] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:48] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:41:48] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: [Errno 60] Operation timed out
[2026-04-13 17:41:48] [Warning]  Attempting to reconnect...
[2026-04-13 17:41:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:41:48] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:48] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:41:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:49] [Extra]    Valid announce for <ec6862efe5c7c99ac944aa1f7ac8f5df> 71 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:41:49] [Debug]    Destination <ec6862efe5c7c99ac944aa1f7ac8f5df> is now 71 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:41:50] [Extra]    Valid announce for <103eb3c7f35278ba33e7d014e341b3ec> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:50] [Debug]    Destination <103eb3c7f35278ba33e7d014e341b3ec> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:50] [Extra]    Valid announce for <103eb3c7f35278ba33e7d014e341b3ec> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:50] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:50] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:50] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:50] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:50] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:50] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:50] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:50] [Debug]    Replacing destination table entry for <e345f6220682e127cab52c3387436778> with new announce, since it was more recently emitted
[2026-04-13 17:41:50] [Debug]    Destination <e345f6220682e127cab52c3387436778> is now 6 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:50] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:50] [Debug]    Destination <02aaf088472435718061211d3752c8ed> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:51] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:51] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:51] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:53] [Extra]    Valid announce for <c987f39c391b4a565a4c585d2da419df> 12 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:41:53] [Debug]    Destination <c987f39c391b4a565a4c585d2da419df> is now 12 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:41:53] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:53] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:41:53] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:53] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:41:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:41:53] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:53] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:41:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:41:53] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:53] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:41:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:53] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:53] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:53] [Debug]    Destination <794884194914d03c4e199d9c1f090b0c> is now 6 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:54] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:55] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:55] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:56] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:56] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:56] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:56] [Extra]    Valid announce for <af1ec9121da534836e6a39b7d261fa65> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:56] [Debug]    Destination <af1ec9121da534836e6a39b7d261fa65> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:56] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:57] [Extra]    Valid announce for <af1ec9121da534836e6a39b7d261fa65> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:41:57] [Extra]    Valid announce for <0e9df50566390f7da1a180806ea7459a> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:41:57] [Extra]    Remembering ratchet <47ed9564cd4c874373c3> for <0e9df50566390f7da1a180806ea7459a>
[2026-04-13 17:41:57] [Debug]    Destination <0e9df50566390f7da1a180806ea7459a> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:41:57] [Extra]    Valid announce for <ff41470c0c58afeb129103a5753bbc0f> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:57] [Debug]    Destination <ff41470c0c58afeb129103a5753bbc0f> is now 6 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:57] [Extra]    Valid announce for <0e9df50566390f7da1a180806ea7459a> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:41:57] [Extra]    Valid announce for <af1ec9121da534836e6a39b7d261fa65> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:57] [Extra]    Valid announce for <0e9df50566390f7da1a180806ea7459a> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:41:58] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:58] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:41:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:41:58] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:58] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:41:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:41:58] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:41:58] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:41:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:41:59] [Debug]    Path request for <41165bf22801b29880cdf766ed8cceaf> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:41:59] [Debug]    Ignoring path request for <41165bf22801b29880cdf766ed8cceaf> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:41:59] [Debug]    Path request for <a0a0a61a0adff637f11e8c75f773d2f4> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:41:59] [Debug]    Ignoring path request for <a0a0a61a0adff637f11e8c75f773d2f4> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:41:59] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:41:59] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:00] [Extra]    Valid announce for <833ac927093e8d33370bed3655844587> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:00] [Debug]    Destination <833ac927093e8d33370bed3655844587> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:00] [Extra]    Valid announce for <833ac927093e8d33370bed3655844587> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:42:03] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:03] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:42:03] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:03] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:42:03] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:03] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:42:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:04] [Debug]    Path request for <a3dfea289534e4b2d0f5730310eebd99> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:04] [Debug]    Ignoring path request for <a3dfea289534e4b2d0f5730310eebd99> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:42:04] [Debug]    Path request for <cf4ca0a1cf91f87778b3543586f75d9f> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:04] [Debug]    Ignoring path request for <cf4ca0a1cf91f87778b3543586f75d9f> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:42:06] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:06] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:06] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:06] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:07] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:07] [Debug]    Destination <192eed7af8e3311445372f2a43cb63ec> is now 2 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:07] [Extra]    Valid announce for <90b625aada641de7f787d21793a546e5> 2 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:07] [Debug]    Destination <90b625aada641de7f787d21793a546e5> is now 2 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:08] [Extra]    Valid announce for <90b625aada641de7f787d21793a546e5> 2 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:42:08] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:08] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:42:08] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:08] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:42:08] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:08] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:42:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:10] [Extra]    Valid announce for <73603b663251f2af83bd09d47919cc7c> 24 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:10] [Debug]    Destination <73603b663251f2af83bd09d47919cc7c> is now 24 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:11] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:11] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:42:13] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:13] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:42:13] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:13] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:42:13] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:13] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:42:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:13] [Extra]    Valid announce for <a430b813dd5c253002380cda46bf8a05> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:13] [Extra]    Remembering ratchet <1fcd72d36900f9673261> for <a430b813dd5c253002380cda46bf8a05>
[2026-04-13 17:42:13] [Debug]    Destination <a430b813dd5c253002380cda46bf8a05> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:13] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:13] [Extra]    Valid announce for <a576f1d475307a39dc9dcbf02e5bcae6> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:13] [Debug]    Destination <a576f1d475307a39dc9dcbf02e5bcae6> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:13] [Extra]    Valid announce for <833ac927093e8d33370bed3655844587> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:13] [Extra]    Valid announce for <6ba4df8ec7f814fc963347d9ee2a4b8e> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:13] [Debug]    Destination <6ba4df8ec7f814fc963347d9ee2a4b8e> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:13] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:13] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:13] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:13] [Extra]    Valid announce for <90b625aada641de7f787d21793a546e5> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:14] [Extra]    Valid announce for <943462afbda6b1f6578b5cdbcedbda07> 6 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:14] [Debug]    Destination <943462afbda6b1f6578b5cdbcedbda07> is now 6 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:14] [Extra]    Valid announce for <9a3437dc85f167c952dfc7e0a7703df3> 6 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:14] [Debug]    Destination <9a3437dc85f167c952dfc7e0a7703df3> is now 6 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:14] [Extra]    Valid announce for <70c584e468e39d5e46e7611d68764a5e> 8 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:14] [Debug]    Destination <70c584e468e39d5e46e7611d68764a5e> is now 8 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:14] [Extra]    Valid announce for <e916365766702b768c93c28dd0caf168> 9 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:14] [Debug]    Destination <e916365766702b768c93c28dd0caf168> is now 9 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:14] [Extra]    Valid announce for <9a91630046262f3537aa9ee595bd02f1> 7 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:14] [Debug]    Destination <9a91630046262f3537aa9ee595bd02f1> is now 7 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:18] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:18] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:42:18] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:18] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:42:18] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:18] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:42:18] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:18] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:42:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:18] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:20] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:20] [Debug]    Destination <02aaf088472435718061211d3752c8ed> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:20] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:20] [Debug]    Destination <e345f6220682e127cab52c3387436778> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:20] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:20] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:20] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:20] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:20] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:20] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:21] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:21] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:21] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:21] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:21] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:21] [Debug]    Path request for <995cc3851347f138f51af73d182d5e1d> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:21] [Debug]    Ignoring path request for <995cc3851347f138f51af73d182d5e1d> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:42:23] [Debug]    Path request for <26211d998926dd810facde56fe28c78f> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:23] [Debug]    Ignoring path request for <26211d998926dd810facde56fe28c78f> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242], no path known
[2026-04-13 17:42:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:42:23] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:23] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:42:23] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:23] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:42:23] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:23] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:42:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:24] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:24] [Debug]    Destination <794884194914d03c4e199d9c1f090b0c> is now 6 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:25] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:26] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:26] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:26] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:26] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:26] [Debug]    Replacing destination table entry for <73400f494c8d580bd774443a5163127b> with new announce, since it was more recently emitted
[2026-04-13 17:42:26] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 5 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:27] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:27] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:42:28] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:28] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:42:28] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:28] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:42:28] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:28] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:42:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:29] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:29] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:30] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:32] [Debug]    Path request for <41ea684bc8b1dc32d23e0d04e9029e1b> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:32] [Debug]    Ignoring path request for <41ea684bc8b1dc32d23e0d04e9029e1b> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:42:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:42:33] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:33] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:42:33] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:33] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:42:33] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:33] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:42:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:34] [Debug]    Path request for <1ba3f953b8e28bd2f5a5ec2e741edf65> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:34] [Debug]    Ignoring path request for <1ba3f953b8e28bd2f5a5ec2e741edf65> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242], no path known
[2026-04-13 17:42:35] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:35] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:35] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:35] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:36] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:36] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:42:38] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:38] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:42:38] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:38] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:42:38] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:38] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:42:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:39] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:39] [Debug]    Destination <e80bd281bd26e00d735aada7b7b94c7a> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:39] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:40] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:40] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:40] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:42] [Debug]    Path request for <f5122ab9149c8794f771c70906bd4705> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:42] [Debug]    Ignoring path request for <f5122ab9149c8794f771c70906bd4705> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:42:42] [Debug]    Path request for <34b2d5e6e88abd17de1616d24ab7a66f> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:42] [Debug]    Ignoring path request for <34b2d5e6e88abd17de1616d24ab7a66f> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:42:42] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:42] [Debug]    Destination <29b2ebe588859e48aabf13e97cfe245b> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:43] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:43] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:43] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:43] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:42:43] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:43] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:42:43] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:43] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:42:43] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:43] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:42:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:46] [Debug]    Path request for <41165bf22801b29880cdf766ed8cceaf> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:46] [Debug]    Ignoring path request for <41165bf22801b29880cdf766ed8cceaf> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:42:47] [Debug]    Path request for <eae49b952fed85ac694a6896bda42e4b> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:47] [Debug]    Ignoring path request for <eae49b952fed85ac694a6896bda42e4b> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:42:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:42:48] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:48] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:42:48] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:48] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:42:48] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:48] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:42:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:49] [Extra]    Valid announce for <103eb3c7f35278ba33e7d014e341b3ec> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:49] [Debug]    Destination <103eb3c7f35278ba33e7d014e341b3ec> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:49] [Extra]    Valid announce for <103eb3c7f35278ba33e7d014e341b3ec> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:49] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:49] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:50] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:50] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:50] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:50] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:50] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:50] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:50] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:50] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:52] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:52] [Debug]    Destination <794884194914d03c4e199d9c1f090b0c> is now 6 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:42:53] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:53] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:42:53] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:53] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:42:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:42:53] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:53] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:53] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:54] [Extra]    Valid announce for <6f5ed4f09288c73e7c60ee96201c9ead> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:54] [Debug]    Destination <6f5ed4f09288c73e7c60ee96201c9ead> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:54] [Extra]    Valid announce for <5da0f016955580a0cad630bd440e7a5a> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:54] [Debug]    Destination <5da0f016955580a0cad630bd440e7a5a> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:54] [Extra]    Valid announce for <d4e654d7fea48a49d8889be71103f52c> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:54] [Extra]    Remembering ratchet <e0e8787d2456d38a42ad> for <d4e654d7fea48a49d8889be71103f52c>
[2026-04-13 17:42:54] [Debug]    Destination <d4e654d7fea48a49d8889be71103f52c> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:54] [Extra]    Valid announce for <6f5ed4f09288c73e7c60ee96201c9ead> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:55] [Extra]    Valid announce for <55d5de5349ba83aa440c17210c44aaab> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:55] [Extra]    Remembering ratchet <9ed18f8afae4ab6969d3> for <55d5de5349ba83aa440c17210c44aaab>
[2026-04-13 17:42:55] [Debug]    Destination <55d5de5349ba83aa440c17210c44aaab> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:55] [Extra]    Valid announce for <7bd7df0c83d109d89ab484faee04a9af> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:55] [Extra]    Remembering ratchet <d3998a2c1e2a12e42839> for <7bd7df0c83d109d89ab484faee04a9af>
[2026-04-13 17:42:55] [Debug]    Destination <7bd7df0c83d109d89ab484faee04a9af> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:55] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:55] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:55] [Extra]    Valid announce for <5da0f016955580a0cad630bd440e7a5a> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:55] [Extra]    Valid announce for <d4e654d7fea48a49d8889be71103f52c> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:55] [Extra]    Valid announce for <7bd7df0c83d109d89ab484faee04a9af> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:56] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:56] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:56] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:56] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:56] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:56] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:56] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:56] [Extra]    Valid announce for <af1ec9121da534836e6a39b7d261fa65> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:56] [Debug]    Destination <af1ec9121da534836e6a39b7d261fa65> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:57] [Extra]    Valid announce for <af1ec9121da534836e6a39b7d261fa65> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:57] [Extra]    Valid announce for <af1ec9121da534836e6a39b7d261fa65> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:58] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:58] [Debug]    Replacing destination table entry for <02aaf088472435718061211d3752c8ed> with new announce, since it was more recently emitted
[2026-04-13 17:42:58] [Debug]    Destination <02aaf088472435718061211d3752c8ed> is now 5 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:42:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:42:58] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:58] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:42:58] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:58] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:42:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:42:58] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:42:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:42:58] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:42:58] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:42:58] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:59] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:42:59] [Debug]    Replacing destination table entry for <e345f6220682e127cab52c3387436778> with new announce, since it was more recently emitted
[2026-04-13 17:42:59] [Debug]    Destination <e345f6220682e127cab52c3387436778> is now 5 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:00] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:00] [Extra]    Valid announce for <a430b813dd5c253002380cda46bf8a05> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:00] [Debug]    Destination <a430b813dd5c253002380cda46bf8a05> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:00] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:00] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:01] [Extra]    Valid announce for <a430b813dd5c253002380cda46bf8a05> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:01] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:01] [Debug]    Path request for <f5122ab9149c8794f771c70906bd4705> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:01] [Debug]    Ignoring path request for <f5122ab9149c8794f771c70906bd4705> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242], no path known
[2026-04-13 17:43:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:43:03] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:03] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:43:03] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:03] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:43:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:43:03] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:03] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:05] [Debug]    Path request for <34b2d5e6e88abd17de1616d24ab7a66f> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:05] [Debug]    Ignoring path request for <34b2d5e6e88abd17de1616d24ab7a66f> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:43:06] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:06] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:06] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:06] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:06] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:06] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:07] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:07] [Debug]    Destination <192eed7af8e3311445372f2a43cb63ec> is now 2 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:08] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:08] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:08] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:08] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:43:08] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:08] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:43:08] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:08] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:43:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:43:08] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:08] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:09] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:11] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:11] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:11] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:12] [Debug]    Path request for <9340787e35f6412f3dbe046693e81589> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:12] [Debug]    Ignoring path request for <9340787e35f6412f3dbe046693e81589> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:43:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:43:13] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:13] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:43:13] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:13] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:43:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:43:13] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:13] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:15] [Debug]    Path request for <fad4b2d1036316292c7a7c92b34c246e> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:15] [Debug]    Ignoring path request for <fad4b2d1036316292c7a7c92b34c246e> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:43:15] [Extra]    Valid announce for <6d1319b8154d542d84fee89c9a131c98> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:15] [Debug]    Destination <6d1319b8154d542d84fee89c9a131c98> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:16] [Extra]    Valid announce for <f96951db4137fcaaab8f2b848aaf5e22> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:16] [Debug]    Destination <f96951db4137fcaaab8f2b848aaf5e22> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:16] [Extra]    Valid announce for <6d1319b8154d542d84fee89c9a131c98> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:16] [Extra]    Valid announce for <f96951db4137fcaaab8f2b848aaf5e22> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:17] [Extra]    Valid announce for <95ca807f05d258f7723d5f1f75c29159> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:17] [Debug]    Destination <95ca807f05d258f7723d5f1f75c29159> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:17] [Extra]    Valid announce for <a242915187d12f5b6d5072165fba4b5d> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:17] [Debug]    Destination <a242915187d12f5b6d5072165fba4b5d> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:17] [Extra]    Valid announce for <ffc8d3472451090677fd446837e384ff> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:17] [Debug]    Destination <ffc8d3472451090677fd446837e384ff> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:18] [Debug]    Path request for <a3dfea289534e4b2d0f5730310eebd99> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:18] [Debug]    Ignoring path request for <a3dfea289534e4b2d0f5730310eebd99> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242], no path known
[2026-04-13 17:43:18] [Debug]    Ignoring duplicate path request for <a3dfea289534e4b2d0f5730310eebd99> with tag <a3dfea289534e4b2d0f5730310eebd99a6f6f8bcdcc48f342b07d39c6f06c4a8>
[2026-04-13 17:43:18] [Extra]    Valid announce for <ffc8d3472451090677fd446837e384ff> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:18] [Extra]    Valid announce for <a8a54ef3254cfe3369383ca34de3a423> 6 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:18] [Debug]    Destination <a8a54ef3254cfe3369383ca34de3a423> is now 6 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:18] [Extra]    Valid announce for <ffc8d3472451090677fd446837e384ff> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:18] [Extra]    Valid announce for <2717faeb3405187e45fecb2bfbab9d4d> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:18] [Debug]    Destination <2717faeb3405187e45fecb2bfbab9d4d> is now 6 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:18] [Extra]    Valid announce for <e99770467e3a33c9f0a4ed8a95e2581e> 7 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:18] [Debug]    Destination <e99770467e3a33c9f0a4ed8a95e2581e> is now 7 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:43:18] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:18] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:43:18] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:18] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:43:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:43:18] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:18] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:20] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:20] [Debug]    Destination <02aaf088472435718061211d3752c8ed> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:20] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:20] [Debug]    Destination <e345f6220682e127cab52c3387436778> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:20] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:20] [Debug]    Destination <794884194914d03c4e199d9c1f090b0c> is now 6 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:20] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:20] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:20] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:20] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:20] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:20] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:21] [Extra]    Valid announce for <d7881baf17ece4f8683923d9b1df6f48> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:21] [Debug]    Destination <d7881baf17ece4f8683923d9b1df6f48> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:23] [Extra]    Valid announce for <219a60c23a74cf1ede2ee1c56dc790d7> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:23] [Extra]    Remembering ratchet <8afca01e29302bf060d5> for <219a60c23a74cf1ede2ee1c56dc790d7>
[2026-04-13 17:43:23] [Debug]    Destination <219a60c23a74cf1ede2ee1c56dc790d7> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:23] [Extra]    Valid announce for <219a60c23a74cf1ede2ee1c56dc790d7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:43:23] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:23] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:43:23] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:23] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:43:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:43:23] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:23] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:24] [Extra]    Valid announce for <d4b95fc3ffe1d8d0bbd9e2da7b0f6157> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:24] [Debug]    Destination <d4b95fc3ffe1d8d0bbd9e2da7b0f6157> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:24] [Extra]    Valid announce for <d4b95fc3ffe1d8d0bbd9e2da7b0f6157> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:24] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:24] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:24] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:24] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:24] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:25] [Extra]    Valid announce for <3e0be866586a077ce39e9e4e065170a1> 41 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:25] [Debug]    Destination <3e0be866586a077ce39e9e4e065170a1> is now 41 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:25] [Extra]    Valid announce for <cb49f32acdc2cc5f8fd5f0cb2ee2ec2e> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:25] [Debug]    Destination <cb49f32acdc2cc5f8fd5f0cb2ee2ec2e> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:25] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:25] [Extra]    Valid announce for <cb49f32acdc2cc5f8fd5f0cb2ee2ec2e> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:25] [Extra]    Valid announce for <3e0be866586a077ce39e9e4e065170a1> 41 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:25] [Extra]    Valid announce for <d4b95fc3ffe1d8d0bbd9e2da7b0f6157> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:25] [Extra]    Valid announce for <cb49f32acdc2cc5f8fd5f0cb2ee2ec2e> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:26] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:26] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:26] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:26] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:26] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:27] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:28] [Extra]    Valid announce for <e6f631470c948894ff4b4e481f4631af> 94 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:28] [Debug]    Destination <e6f631470c948894ff4b4e481f4631af> is now 94 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:28] [Extra]    Valid announce for <82439e5d8ceaf0fbb8c78b022fb70c0d> 10 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:28] [Debug]    Destination <82439e5d8ceaf0fbb8c78b022fb70c0d> is now 10 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:28] [Extra]    Valid announce for <e6f631470c948894ff4b4e481f4631af> 94 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:43:28] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:28] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:43:28] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:28] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:43:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:43:28] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:28] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:29] [Extra]    Valid announce for <6ee8d89ae74833c397169c07b81e62e2> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:29] [Debug]    Destination <6ee8d89ae74833c397169c07b81e62e2> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:30] [Extra]    Valid announce for <8976c1b2ae6b60fd1a09a83a6e64ff93> 7 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:30] [Debug]    Destination <8976c1b2ae6b60fd1a09a83a6e64ff93> is now 7 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:30] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:30] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:30] [Extra]    Valid announce for <8976c1b2ae6b60fd1a09a83a6e64ff93> 7 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:30] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:32] [Extra]    Valid announce for <8976c1b2ae6b60fd1a09a83a6e64ff93> 9 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:43:33] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:33] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:43:33] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:33] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:43:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:43:33] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:33] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:33] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:33] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:34] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:35] [Extra]    Valid announce for <3e0be866586a077ce39e9e4e065170a1> 43 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:35] [Extra]    Valid announce for <e6f631470c948894ff4b4e481f4631af> 95 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:36] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:36] [Extra]    Valid announce for <e6f631470c948894ff4b4e481f4631af> 96 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:36] [Debug]    Path request for <995cc3851347f138f51af73d182d5e1d> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:36] [Debug]    Ignoring path request for <995cc3851347f138f51af73d182d5e1d> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:43:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:43:38] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:38] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:43:38] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:38] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:43:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:43:38] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:38] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:38] [Debug]    Path request for <41165bf22801b29880cdf766ed8cceaf> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:38] [Debug]    Ignoring path request for <41165bf22801b29880cdf766ed8cceaf> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:43:38] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:38] [Debug]    Destination <e80bd281bd26e00d735aada7b7b94c7a> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:41] [Extra]    Valid announce for <afd9e647b556572d78d8b36e3390000b> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:41] [Extra]    Remembering ratchet <0b0fa337a6cfe47714d7> for <afd9e647b556572d78d8b36e3390000b>
[2026-04-13 17:43:41] [Debug]    Destination <afd9e647b556572d78d8b36e3390000b> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:41] [Extra]    Valid announce for <8976c1b2ae6b60fd1a09a83a6e64ff93> 11 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:41] [Extra]    Valid announce for <c03bcc5418d9a06050e23c7daa68d5b2> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:41] [Debug]    Destination <c03bcc5418d9a06050e23c7daa68d5b2> is now 5 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:41] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:41] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:41] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:41] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:41] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:41] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:42] [Extra]    Valid announce for <58cea53bfb32988291b49a6205388cd1> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:42] [Extra]    Remembering ratchet <371466d8bf532057a086> for <58cea53bfb32988291b49a6205388cd1>
[2026-04-13 17:43:42] [Debug]    Destination <58cea53bfb32988291b49a6205388cd1> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:42] [Extra]    Valid announce for <afd9e647b556572d78d8b36e3390000b> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:42] [Extra]    Valid announce for <c03bcc5418d9a06050e23c7daa68d5b2> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:42] [Extra]    Valid announce for <58cea53bfb32988291b49a6205388cd1> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:42] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:42] [Debug]    Destination <29b2ebe588859e48aabf13e97cfe245b> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:43] [Extra]    Valid announce for <b36d8cabbd9eb695012558a30df33963> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:43] [Debug]    Destination <b36d8cabbd9eb695012558a30df33963> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:43] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:43] [Extra]    Valid announce for <ea4a5a9a01c9ad6e39718716e6cf9c06> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:43] [Debug]    Replacing destination table entry for <ea4a5a9a01c9ad6e39718716e6cf9c06> with new announce, since it was more recently emitted
[2026-04-13 17:43:43] [Debug]    Destination <ea4a5a9a01c9ad6e39718716e6cf9c06> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:43] [Extra]    Valid announce for <b36d8cabbd9eb695012558a30df33963> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:43:43] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:43] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:43:43] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:43] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:43:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:43:43] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:43] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:47] [Extra]    Valid announce for <8976c1b2ae6b60fd1a09a83a6e64ff93> 11 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:48] [Extra]    Valid announce for <3d670edfc4940393c314db7eb05460f7> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:48] [Extra]    Remembering ratchet <627e676193bba676d363> for <3d670edfc4940393c314db7eb05460f7>
[2026-04-13 17:43:48] [Debug]    Destination <3d670edfc4940393c314db7eb05460f7> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:43:48] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:48] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:43:48] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:48] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:43:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:43:48] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:48] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:50] [Extra]    Valid announce for <103eb3c7f35278ba33e7d014e341b3ec> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:50] [Debug]    Replacing destination table entry for <103eb3c7f35278ba33e7d014e341b3ec> with new announce, since it was more recently emitted
[2026-04-13 17:43:50] [Debug]    Destination <103eb3c7f35278ba33e7d014e341b3ec> is now 6 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:51] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:51] [Debug]    Replacing destination table entry for <02aaf088472435718061211d3752c8ed> with new announce, since it was more recently emitted
[2026-04-13 17:43:51] [Debug]    Destination <02aaf088472435718061211d3752c8ed> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:51] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:51] [Debug]    Replacing destination table entry for <e345f6220682e127cab52c3387436778> with new announce, since it was more recently emitted
[2026-04-13 17:43:51] [Debug]    Destination <e345f6220682e127cab52c3387436778> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:51] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:51] [Debug]    Replacing destination table entry for <2d8a25919ea488ce008d3635d9b104c7> with new announce, since it was more recently emitted
[2026-04-13 17:43:51] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:51] [Extra]    Valid announce for <e7bbfebb0d6f5ad15e2a96dd3200d215> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:51] [Extra]    Remembering ratchet <f86710f5d913eed6fe6b> for <e7bbfebb0d6f5ad15e2a96dd3200d215>
[2026-04-13 17:43:51] [Debug]    Destination <e7bbfebb0d6f5ad15e2a96dd3200d215> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:51] [Extra]    Valid announce for <f9590b10757da33e1953feaa32912373> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:51] [Debug]    Destination <f9590b10757da33e1953feaa32912373> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:52] [Extra]    Valid announce for <e7bbfebb0d6f5ad15e2a96dd3200d215> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:52] [Extra]    Valid announce for <f9590b10757da33e1953feaa32912373> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:52] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:52] [Debug]    Destination <794884194914d03c4e199d9c1f090b0c> is now 6 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:43:53] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:53] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:43:53] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:53] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:43:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:43:53] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:53] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:54] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <afd9e647b556572d78d8b36e3390000b> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <58cea53bfb32988291b49a6205388cd1> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <c03bcc5418d9a06050e23c7daa68d5b2> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <ea4a5a9a01c9ad6e39718716e6cf9c06> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <b36d8cabbd9eb695012558a30df33963> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <3d670edfc4940393c314db7eb05460f7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <103eb3c7f35278ba33e7d014e341b3ec> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <e7bbfebb0d6f5ad15e2a96dd3200d215> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <f9590b10757da33e1953feaa32912373> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:54] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:55] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:55] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:55] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:55] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:55] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:55] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:55] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:55] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:56] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:56] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:56] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:56] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:56] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:56] [Extra]    Valid announce for <0019bfdaad8067b50f13c5342d1e7b16> 27 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:56] [Debug]    Destination <0019bfdaad8067b50f13c5342d1e7b16> is now 27 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:56] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:56] [Extra]    Valid announce for <af1ec9121da534836e6a39b7d261fa65> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:56] [Debug]    Destination <af1ec9121da534836e6a39b7d261fa65> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:57] [Extra]    Valid announce for <af1ec9121da534836e6a39b7d261fa65> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:43:58] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:58] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:43:58] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:58] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:43:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:43:58] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:43:58] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:43:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:43:58] [Extra]    Valid announce for <29c93015bace116dbb02fc0f5cbf1d9b> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:58] [Debug]    Destination <29c93015bace116dbb02fc0f5cbf1d9b> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:43:59] [Extra]    Valid announce for <29c93015bace116dbb02fc0f5cbf1d9b> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:59] [Extra]    Valid announce for <9bb76650536202dc4e26313698e35a61> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:59] [Debug]    Destination <9bb76650536202dc4e26313698e35a61> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:59] [Extra]    Valid announce for <3b171e0b79acf468ae1bf3a6d8515d12> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:59] [Debug]    Destination <3b171e0b79acf468ae1bf3a6d8515d12> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:43:59] [Extra]    Valid announce for <3b171e0b79acf468ae1bf3a6d8515d12> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:59] [Extra]    Valid announce for <29c93015bace116dbb02fc0f5cbf1d9b> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:43:59] [Extra]    Valid announce for <9bb76650536202dc4e26313698e35a61> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:00] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 8 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:01] [Extra]    Valid announce for <a430b813dd5c253002380cda46bf8a05> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:01] [Debug]    Destination <a430b813dd5c253002380cda46bf8a05> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:01] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:01] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:01] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:01] [Extra]    Valid announce for <a430b813dd5c253002380cda46bf8a05> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:03] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:03] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:44:03] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:03] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:44:03] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:03] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:44:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:44:03] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:03] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:03] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:07] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:07] [Debug]    Destination <192eed7af8e3311445372f2a43cb63ec> is now 2 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:08] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:44:08] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:08] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:44:08] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:08] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:44:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:44:08] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:08] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:09] [Extra]    Valid announce for <3b171e0b79acf468ae1bf3a6d8515d12> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:09] [Debug]    Replacing destination table entry for <3b171e0b79acf468ae1bf3a6d8515d12> with new announce, since it was more recently emitted
[2026-04-13 17:44:09] [Debug]    Destination <3b171e0b79acf468ae1bf3a6d8515d12> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:09] [Debug]    Path request for <4cce8a55cc0f232fb0946b392a73fa92> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:44:09] [Debug]    Ignoring path request for <4cce8a55cc0f232fb0946b392a73fa92> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:44:10] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:10] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:10] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:10] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:11] [Extra]    Valid announce for <3b171e0b79acf468ae1bf3a6d8515d12> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:11] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:11] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:11] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:11] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:12] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:44:13] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:13] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:44:13] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:13] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:44:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:44:13] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:13] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:15] [Extra]    Valid announce for <3b171e0b79acf468ae1bf3a6d8515d12> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:15] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 7 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:15] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 7 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:16] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 7 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:16] [Extra]    Valid announce for <95ca807f05d258f7723d5f1f75c29159> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:17] [Debug]    Destination <95ca807f05d258f7723d5f1f75c29159> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:17] [Extra]    Valid announce for <af1ec9121da534836e6a39b7d261fa65> 7 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:17] [Extra]    Valid announce for <95ca807f05d258f7723d5f1f75c29159> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:18] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:18] [Extra]    Valid announce for <ffc8d3472451090677fd446837e384ff> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:18] [Debug]    Destination <ffc8d3472451090677fd446837e384ff> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:44:18] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:18] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:44:18] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:18] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:44:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:44:18] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:18] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:19] [Extra]    Valid announce for <ffc8d3472451090677fd446837e384ff> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:19] [Extra]    Valid announce for <dc1665cfd1f79fb83b430d953bb13f59> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:21] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 8 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:21] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:21] [Debug]    Destination <02aaf088472435718061211d3752c8ed> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:21] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:21] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:21] [Debug]    Destination <e345f6220682e127cab52c3387436778> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:21] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:21] [Debug]    Replacing destination table entry for <2d8a25919ea488ce008d3635d9b104c7> with new announce, since it was more recently emitted
[2026-04-13 17:44:21] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:21] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 5 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:22] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:22] [Extra]    Valid announce for <38073923c15b25893cd38a7938a9943a> 7 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:22] [Extra]    Remembering ratchet <90fecec03e5de4bb72d9> for <38073923c15b25893cd38a7938a9943a>
[2026-04-13 17:44:22] [Debug]    Destination <38073923c15b25893cd38a7938a9943a> is now 7 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:23] [Extra]    Valid announce for <219a60c23a74cf1ede2ee1c56dc790d7> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:23] [Debug]    Destination <219a60c23a74cf1ede2ee1c56dc790d7> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:23] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:23] [Debug]    Destination <794884194914d03c4e199d9c1f090b0c> is now 6 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:23] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:44:23] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:23] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:44:23] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:23] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:44:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:44:23] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:23] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:24] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:24] [Debug]    Replacing destination table entry for <110d7f3159c1d306851c3ec5c6d302ef> with new announce, since it was more recently emitted
[2026-04-13 17:44:24] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:24] [Extra]    Valid announce for <a242915187d12f5b6d5072165fba4b5d> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:24] [Debug]    Destination <a242915187d12f5b6d5072165fba4b5d> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:24] [Extra]    Valid announce for <219a60c23a74cf1ede2ee1c56dc790d7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:24] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:25] [Extra]    Valid announce for <a242915187d12f5b6d5072165fba4b5d> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:25] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:25] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:25] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:25] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:25] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:25] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:26] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:26] [Debug]    Replacing destination table entry for <ca273d664d1a6c59a5a002670a641eff> with new announce, since it was more recently emitted
[2026-04-13 17:44:26] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:26] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:26] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:26] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:27] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:28] [Extra]    Valid announce for <6ee8d89ae74833c397169c07b81e62e2> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:28] [Debug]    Destination <6ee8d89ae74833c397169c07b81e62e2> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:28] [Extra]    Valid announce for <6ee8d89ae74833c397169c07b81e62e2> 2 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:44:28] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:28] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:44:28] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:28] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:44:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:44:28] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:28] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:30] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:30] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:31] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:44:33] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:33] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:44:33] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:33] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:44:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:44:33] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:33] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:33] [Debug]    Path request for <a3dfea289534e4b2d0f5730310eebd99> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:44:33] [Debug]    Ignoring path request for <a3dfea289534e4b2d0f5730310eebd99> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:44:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:44:38] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:38] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:44:38] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:38] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:44:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:44:38] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:38] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:38] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:38] [Debug]    Destination <e80bd281bd26e00d735aada7b7b94c7a> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:39] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:40] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:40] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:40] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:40] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:40] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:40] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:41] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:41] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:41] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:42] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:42] [Debug]    Destination <29b2ebe588859e48aabf13e97cfe245b> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:43] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:44:43] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:43] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:44:43] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:43] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:44:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:44:43] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:43] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:46] [Extra]    Valid announce for <58cea53bfb32988291b49a6205388cd1> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:46] [Debug]    Replacing destination table entry for <58cea53bfb32988291b49a6205388cd1> with new announce, since it was more recently emitted
[2026-04-13 17:44:46] [Debug]    Destination <58cea53bfb32988291b49a6205388cd1> is now 5 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:46] [Extra]    Valid announce for <c03bcc5418d9a06050e23c7daa68d5b2> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:46] [Debug]    Destination <c03bcc5418d9a06050e23c7daa68d5b2> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:46] [Extra]    Valid announce for <d7881baf17ece4f8683923d9b1df6f48> 7 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:46] [Extra]    Valid announce for <58cea53bfb32988291b49a6205388cd1> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:46] [Extra]    Valid announce for <c03bcc5418d9a06050e23c7daa68d5b2> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:48] [Extra]    Valid announce for <103eb3c7f35278ba33e7d014e341b3ec> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:48] [Debug]    Destination <103eb3c7f35278ba33e7d014e341b3ec> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:44:48] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:48] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:44:48] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:48] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:44:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:44:48] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:48] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:48] [Extra]    Valid announce for <103eb3c7f35278ba33e7d014e341b3ec> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:49] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:49] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:49] [Extra]    Valid announce for <3d670edfc4940393c314db7eb05460f7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:49] [Debug]    Replacing destination table entry for <3d670edfc4940393c314db7eb05460f7> with new announce, since it was more recently emitted
[2026-04-13 17:44:49] [Debug]    Destination <3d670edfc4940393c314db7eb05460f7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:49] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:49] [Extra]    Valid announce for <3d670edfc4940393c314db7eb05460f7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:50] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:50] [Debug]    Destination <02aaf088472435718061211d3752c8ed> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:50] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:50] [Debug]    Destination <e345f6220682e127cab52c3387436778> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:50] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:50] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:51] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:51] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:51] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:51] [Debug]    Destination <794884194914d03c4e199d9c1f090b0c> is now 6 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:51] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:51] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 7 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:53] [Debug]    Path request for <995cc3851347f138f51af73d182d5e1d> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:44:53] [Debug]    Ignoring path request for <995cc3851347f138f51af73d182d5e1d> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:44:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:44:53] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:53] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:44:53] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:53] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:44:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:44:53] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:53] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:54] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:54] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:55] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:55] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:55] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:55] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:56] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:56] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:56] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:56] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:56] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:56] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:57] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:57] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:57] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:57] [Extra]    Valid announce for <0e9df50566390f7da1a180806ea7459a> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:57] [Debug]    Destination <0e9df50566390f7da1a180806ea7459a> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:57] [Extra]    Valid announce for <af1ec9121da534836e6a39b7d261fa65> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:57] [Debug]    Destination <af1ec9121da534836e6a39b7d261fa65> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:58] [Extra]    Valid announce for <0e9df50566390f7da1a180806ea7459a> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:58] [Extra]    Valid announce for <af1ec9121da534836e6a39b7d261fa65> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:44:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:44:58] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:58] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:44:58] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:58] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:44:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:44:58] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:44:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:44:58] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:44:59] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:44:59] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:00] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:01] [Extra]    Valid announce for <e6f631470c948894ff4b4e481f4631af> 93 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:01] [Extra]    Valid announce for <a430b813dd5c253002380cda46bf8a05> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:01] [Debug]    Replacing destination table entry for <a430b813dd5c253002380cda46bf8a05> with new announce, since it was more recently emitted
[2026-04-13 17:45:01] [Debug]    Destination <a430b813dd5c253002380cda46bf8a05> is now 5 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:01] [Extra]    Valid announce for <6862f26ba0bd11ecb058676e99192762> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:01] [Debug]    Destination <6862f26ba0bd11ecb058676e99192762> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:02] [Extra]    Valid announce for <456a0c7be5d912e51e23183edc77d39a> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:02] [Debug]    Destination <456a0c7be5d912e51e23183edc77d39a> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:02] [Extra]    Valid announce for <456a0c7be5d912e51e23183edc77d39a> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:02] [Extra]    Valid announce for <a430b813dd5c253002380cda46bf8a05> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:02] [Extra]    Valid announce for <6862f26ba0bd11ecb058676e99192762> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:45:03] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:03] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:45:03] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:03] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:45:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:45:03] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:03] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:05] [Extra]    Valid announce for <0ffbe2818af6cd15cb0931bab5f894f0> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:05] [Debug]    Destination <0ffbe2818af6cd15cb0931bab5f894f0> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:05] [Extra]    Valid announce for <ae66591fbcb603812000b2e752838129> 8 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:05] [Debug]    Destination <ae66591fbcb603812000b2e752838129> is now 8 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:05] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:05] [Extra]    Valid announce for <68370ddefb86a6fc5aa142ff69dc08a1> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:05] [Debug]    Destination <68370ddefb86a6fc5aa142ff69dc08a1> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:05] [Extra]    Valid announce for <16ca26cbfe503916ac4a52c8edba5bb1> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:05] [Debug]    Destination <16ca26cbfe503916ac4a52c8edba5bb1> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:05] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 7 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:05] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:05] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:06] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:06] [Extra]    Valid announce for <58cea53bfb32988291b49a6205388cd1> 6 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:06] [Extra]    Valid announce for <c03bcc5418d9a06050e23c7daa68d5b2> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:06] [Extra]    Valid announce for <d7881baf17ece4f8683923d9b1df6f48> 8 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:06] [Extra]    Valid announce for <103eb3c7f35278ba33e7d014e341b3ec> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:06] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:06] [Extra]    Valid announce for <3d670edfc4940393c314db7eb05460f7> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:06] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:06] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:06] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:06] [Extra]    Valid announce for <0e9df50566390f7da1a180806ea7459a> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:06] [Extra]    Valid announce for <e6f631470c948894ff4b4e481f4631af> 94 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:06] [Debug]    Path request for <41165bf22801b29880cdf766ed8cceaf> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:06] [Debug]    Ignoring path request for <41165bf22801b29880cdf766ed8cceaf> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:45:08] [Debug]    Path request for <57a868d333476362e91d5189c3bc47d9> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:08] [Debug]    Ignoring path request for <57a868d333476362e91d5189c3bc47d9> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:45:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:45:08] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:08] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:45:08] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:08] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:45:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:45:08] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:08] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:10] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:10] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:10] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:10] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:10] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:10] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:11] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:11] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:12] [Extra]    Valid announce for <e6f631470c948894ff4b4e481f4631af> 94 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:12] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:12] [Extra]    Valid announce for <29c93015bace116dbb02fc0f5cbf1d9b> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:13] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:13] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:13] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:45:13] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:13] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:45:13] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:13] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:45:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:45:13] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:13] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:14] [Debug]    Path request for <204f895a19ad6717f387a774be3266db> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:14] [Debug]    Ignoring path request for <204f895a19ad6717f387a774be3266db> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:45:16] [Extra]    Valid announce for <a242915187d12f5b6d5072165fba4b5d> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:16] [Debug]    Destination <a242915187d12f5b6d5072165fba4b5d> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:17] [Extra]    Valid announce for <95ca807f05d258f7723d5f1f75c29159> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:17] [Debug]    Destination <95ca807f05d258f7723d5f1f75c29159> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:17] [Extra]    Valid announce for <a242915187d12f5b6d5072165fba4b5d> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:17] [Extra]    Valid announce for <95ca807f05d258f7723d5f1f75c29159> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:17] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:17] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:18] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:18] [Extra]    Valid announce for <ffc8d3472451090677fd446837e384ff> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:18] [Debug]    Destination <ffc8d3472451090677fd446837e384ff> is now 5 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:45:18] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:18] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:45:18] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:18] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:45:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:45:18] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:18] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:19] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:19] [Debug]    Destination <192eed7af8e3311445372f2a43cb63ec> is now 2 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:19] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:19] [Extra]    Valid announce for <ffc8d3472451090677fd446837e384ff> 7 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:19] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:19] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:19] [Extra]    Valid announce for <d7881baf17ece4f8683923d9b1df6f48> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:19] [Debug]    Destination <d7881baf17ece4f8683923d9b1df6f48> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:20] [Extra]    Valid announce for <d7881baf17ece4f8683923d9b1df6f48> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:20] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:20] [Debug]    Destination <e345f6220682e127cab52c3387436778> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:20] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:20] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:21] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:21] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:21] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:21] [Debug]    Replacing destination table entry for <02aaf088472435718061211d3752c8ed> with new announce, since it was more recently emitted
[2026-04-13 17:45:21] [Debug]    Destination <02aaf088472435718061211d3752c8ed> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:21] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:23] [Extra]    Valid announce for <219a60c23a74cf1ede2ee1c56dc790d7> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:23] [Debug]    Destination <219a60c23a74cf1ede2ee1c56dc790d7> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:45:23] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:23] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:45:23] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:23] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:45:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:23] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:45:23] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:23] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:23] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:23] [Extra]    Valid announce for <219a60c23a74cf1ede2ee1c56dc790d7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:24] [Debug]    Path request for <4762234229b7e0b83c68b857bdf245c3> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:24] [Debug]    Ignoring path request for <4762234229b7e0b83c68b857bdf245c3> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242], no path known
[2026-04-13 17:45:24] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:24] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:24] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:24] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:25] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:25] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:25] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:25] [Debug]    Destination <794884194914d03c4e199d9c1f090b0c> is now 6 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:26] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:26] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:26] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:26] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:26] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:26] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:27] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:27] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:28] [Extra]    Valid announce for <6ee8d89ae74833c397169c07b81e62e2> 2 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:28] [Debug]    Destination <6ee8d89ae74833c397169c07b81e62e2> is now 2 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:28] [Extra]    Valid announce for <6ee8d89ae74833c397169c07b81e62e2> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:28] [Extra]    Valid announce for <6ee8d89ae74833c397169c07b81e62e2> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:45:28] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:28] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:45:28] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:28] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:45:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:28] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:45:28] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:28] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:28] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:31] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:31] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:31] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:45:33] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:33] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:45:33] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:33] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:45:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:33] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:45:33] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:33] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:33] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:35] [Extra]    Valid announce for <3b171e0b79acf468ae1bf3a6d8515d12> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:35] [Debug]    Destination <3b171e0b79acf468ae1bf3a6d8515d12> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:35] [Extra]    Valid announce for <3b171e0b79acf468ae1bf3a6d8515d12> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:37] [Extra]    Valid announce for <8976c1b2ae6b60fd1a09a83a6e64ff93> 11 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:38] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:38] [Debug]    Destination <e80bd281bd26e00d735aada7b7b94c7a> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:38] [Extra]    Valid announce for <8976c1b2ae6b60fd1a09a83a6e64ff93> 11 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:38] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:38] [Debug]    Path request for <9dd2d5f7d8c461dc1b6116a5b3caebbf> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:38] [Debug]    Ignoring path request for <9dd2d5f7d8c461dc1b6116a5b3caebbf> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242], no path known
[2026-04-13 17:45:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:45:38] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:38] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:45:38] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:38] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:45:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:38] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:45:38] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:38] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:38] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:38] [Debug]    Path request for <3b171e0b79acf468ae1bf3a6d8515d12> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:38] [Debug]    Ignoring path request for <3b171e0b79acf468ae1bf3a6d8515d12> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242], no path known
[2026-04-13 17:45:39] [Extra]    Valid announce for <e80bd281bd26e00d735aada7b7b94c7a> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:39] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:39] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:40] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:40] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:40] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:40] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:40] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:40] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:41] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:41] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:41] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:42] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:42] [Extra]    Valid announce for <ea4a5a9a01c9ad6e39718716e6cf9c06> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:42] [Debug]    Destination <ea4a5a9a01c9ad6e39718716e6cf9c06> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:42] [Debug]    Path request for <d3bd4df9b985db034f4bc7459b07fa3c> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:42] [Debug]    Ignoring path request for <d3bd4df9b985db034f4bc7459b07fa3c> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:45:42] [Debug]    Path request for <f5122ab9149c8794f771c70906bd4705> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:42] [Debug]    Ignoring path request for <f5122ab9149c8794f771c70906bd4705> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:45:43] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:43] [Debug]    Destination <29b2ebe588859e48aabf13e97cfe245b> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:43] [Extra]    Valid announce for <ea4a5a9a01c9ad6e39718716e6cf9c06> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:43] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:43] [Extra]    Valid announce for <29b2ebe588859e48aabf13e97cfe245b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:45:43] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:43] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:45:43] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:43] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:45:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:43] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:45:43] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:43] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:43] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:44] [Extra]    Valid announce for <8976c1b2ae6b60fd1a09a83a6e64ff93> 11 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:45] [Extra]    Valid announce for <3b171e0b79acf468ae1bf3a6d8515d12> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:45] [Debug]    Replacing destination table entry for <3b171e0b79acf468ae1bf3a6d8515d12> with new announce, since it was more recently emitted
[2026-04-13 17:45:45] [Debug]    Destination <3b171e0b79acf468ae1bf3a6d8515d12> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:46] [Extra]    Valid announce for <3b171e0b79acf468ae1bf3a6d8515d12> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:47] [Extra]    Valid announce for <f9590b10757da33e1953feaa32912373> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:47] [Debug]    Destination <f9590b10757da33e1953feaa32912373> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:47] [Extra]    Valid announce for <f9590b10757da33e1953feaa32912373> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:48] [Extra]    Valid announce for <f9590b10757da33e1953feaa32912373> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:45:48] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:48] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:45:48] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:48] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:45:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:48] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:45:48] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:48] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:48] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:50] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:50] [Debug]    Replacing destination table entry for <e345f6220682e127cab52c3387436778> with new announce, since it was more recently emitted
[2026-04-13 17:45:50] [Debug]    Destination <e345f6220682e127cab52c3387436778> is now 5 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:50] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:50] [Debug]    Destination <02aaf088472435718061211d3752c8ed> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:50] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:50] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:51] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:51] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:51] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:52] [Extra]    Valid announce for <103eb3c7f35278ba33e7d014e341b3ec> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:52] [Debug]    Destination <103eb3c7f35278ba33e7d014e341b3ec> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:53] [Debug]    Path request for <a3dfea289534e4b2d0f5730310eebd99> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:53] [Debug]    Ignoring path request for <a3dfea289534e4b2d0f5730310eebd99> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242], no path known
[2026-04-13 17:45:53] [Debug]    Ignoring duplicate path request for <a3dfea289534e4b2d0f5730310eebd99> with tag <a3dfea289534e4b2d0f5730310eebd99cd7e98294f1319313f38eae78116f86c>
[2026-04-13 17:45:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:45:53] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:53] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:45:53] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:53] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:45:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:53] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:45:53] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:53] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:53] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:53] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:53] [Debug]    Destination <794884194914d03c4e199d9c1f090b0c> is now 6 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:54] [Extra]    Valid announce for <103eb3c7f35278ba33e7d014e341b3ec> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:54] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:55] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:55] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:55] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:55] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:55] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:56] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:56] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:56] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:56] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:56] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:56] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:45:56] [Extra]    Valid announce for <af1ec9121da534836e6a39b7d261fa65> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:56] [Debug]    Destination <af1ec9121da534836e6a39b7d261fa65> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:56] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:57] [Extra]    Valid announce for <af1ec9121da534836e6a39b7d261fa65> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:57] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:57] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:45:57] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:57] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:45:58] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:58] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:45:58] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:58] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:45:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:45:58] [Debug]    Path request for <0d2f747395d7aab3cc57b00330fecf4a> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:45:58] [Debug]    Ignoring path request for <0d2f747395d7aab3cc57b00330fecf4a> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242], no path known
[2026-04-13 17:45:58] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:45:58] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:45:58] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:45:58] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:46:01] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:01] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:01] [Extra]    Valid announce for <75a761de7bdd03adeb6ad16b004e53a1> 9 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:01] [Extra]    Valid announce for <a430b813dd5c253002380cda46bf8a05> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:01] [Debug]    Destination <a430b813dd5c253002380cda46bf8a05> is now 5 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:01] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:01] [Extra]    Valid announce for <a430b813dd5c253002380cda46bf8a05> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:02] [Extra]    Valid announce for <6862f26ba0bd11ecb058676e99192762> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:02] [Debug]    Destination <6862f26ba0bd11ecb058676e99192762> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:02] [Extra]    Valid announce for <6862f26ba0bd11ecb058676e99192762> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:03] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:03] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:03] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:46:03] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:46:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:46:03] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:46:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:46:03] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:46:03] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:46:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:46:03] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:46:03] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:46:03] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:46:03] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:46:04] [Debug]    Path request for <41165bf22801b29880cdf766ed8cceaf> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:04] [Debug]    Ignoring path request for <41165bf22801b29880cdf766ed8cceaf> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:46:06] [Extra]    Valid announce for <2fb8b45523a7aed73e80973854aaa9ec> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:06] [Extra]    Remembering ratchet <945c60a87f9fb89acd52> for <2fb8b45523a7aed73e80973854aaa9ec>
[2026-04-13 17:46:06] [Debug]    Destination <2fb8b45523a7aed73e80973854aaa9ec> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:06] [Extra]    Valid announce for <546c10847829751484d1051182a1bde6> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:06] [Debug]    Destination <546c10847829751484d1051182a1bde6> is now 6 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:06] [Extra]    Valid announce for <2fb8b45523a7aed73e80973854aaa9ec> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:06] [Extra]    Valid announce for <2fb8b45523a7aed73e80973854aaa9ec> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:06] [Debug]    Path request for <995cc3851347f138f51af73d182d5e1d> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:06] [Debug]    Ignoring path request for <995cc3851347f138f51af73d182d5e1d> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:46:07] [Extra]    Valid announce for <2b4ea7044c74fc8cc4843d75d072fc00> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:07] [Debug]    Destination <2b4ea7044c74fc8cc4843d75d072fc00> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:07] [Extra]    Valid announce for <f05ab4a7fd84074d040f2b5336bfe8d9> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:07] [Debug]    Destination <f05ab4a7fd84074d040f2b5336bfe8d9> is now 6 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:07] [Extra]    Valid announce for <bdfbd5c09f84ac36fd07c1d4a16b9519> 8 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:07] [Debug]    Destination <bdfbd5c09f84ac36fd07c1d4a16b9519> is now 8 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:07] [Extra]    Valid announce for <2b4ea7044c74fc8cc4843d75d072fc00> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:07] [Extra]    Valid announce for <68370ddefb86a6fc5aa142ff69dc08a1> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:07] [Debug]    Replacing destination table entry for <68370ddefb86a6fc5aa142ff69dc08a1> with new announce, since it was more recently emitted
[2026-04-13 17:46:07] [Debug]    Destination <68370ddefb86a6fc5aa142ff69dc08a1> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:07] [Extra]    Valid announce for <68370ddefb86a6fc5aa142ff69dc08a1> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:08] [Extra]    Valid announce for <68370ddefb86a6fc5aa142ff69dc08a1> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:08] [Extra]    Valid announce for <ef98bc2981f7d1e793bb391ca0208231> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:08] [Debug]    Destination <ef98bc2981f7d1e793bb391ca0208231> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:08] [Extra]    Valid announce for <ef98bc2981f7d1e793bb391ca0208231> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:46:08] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:46:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:46:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:46:08] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:46:08] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:46:08] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:46:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:46:08] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:46:08] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:46:08] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:46:08] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:46:09] [Extra]    Valid announce for <d7e6c4222f0174b8881e9a47f424fcf8> 7 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:09] [Debug]    Destination <d7e6c4222f0174b8881e9a47f424fcf8> is now 7 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:09] [Extra]    Valid announce for <3151bdd81fa5d564b1fb766784291fed> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:09] [Debug]    Destination <3151bdd81fa5d564b1fb766784291fed> is now 6 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:09] [Debug]    Path request for <9340787e35f6412f3dbe046693e81589> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:09] [Debug]    Ignoring path request for <9340787e35f6412f3dbe046693e81589> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-13 17:46:10] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:10] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:10] [Extra]    Valid announce for <d5d6cdb6d43cea76db89a50c6a7c3bcf> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:10] [Debug]    Destination <d5d6cdb6d43cea76db89a50c6a7c3bcf> is now 6 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:10] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:10] [Debug]    Destination <ca273d664d1a6c59a5a002670a641eff> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:10] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:10] [Debug]    Destination <73400f494c8d580bd774443a5163127b> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:10] [Extra]    Valid announce for <ca273d664d1a6c59a5a002670a641eff> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:10] [Extra]    Valid announce for <73400f494c8d580bd774443a5163127b> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:10] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:10] [Extra]    Valid announce for <6ea9ba935598917da4dd305962c2760c> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:10] [Debug]    Destination <6ea9ba935598917da4dd305962c2760c> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:11] [Extra]    Valid announce for <f8a4bdb249dbe18c83e254b52edad748> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:11] [Debug]    Destination <f8a4bdb249dbe18c83e254b52edad748> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:11] [Extra]    Valid announce for <6ea9ba935598917da4dd305962c2760c> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:11] [Extra]    Valid announce for <f8a4bdb249dbe18c83e254b52edad748> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:11] [Extra]    Valid announce for <f0921573abf41194ce99f3976f3c7792> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:11] [Debug]    Destination <f0921573abf41194ce99f3976f3c7792> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:12] [Extra]    Valid announce for <82439e5d8ceaf0fbb8c78b022fb70c0d> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:12] [Debug]    Destination <82439e5d8ceaf0fbb8c78b022fb70c0d> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:12] [Extra]    Valid announce for <82439e5d8ceaf0fbb8c78b022fb70c0d> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:13] [Extra]    Valid announce for <82439e5d8ceaf0fbb8c78b022fb70c0d> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:46:13] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:46:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:46:13] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:46:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:46:13] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:46:13] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:46:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:46:13] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:46:13] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:46:13] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:46:13] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:46:13] [Extra]    Valid announce for <0b1eaf702382e3ce5d4a5ab241240b21> 8 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:13] [Debug]    Destination <0b1eaf702382e3ce5d4a5ab241240b21> is now 8 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:15] [Extra]    Valid announce for <d7e6c4222f0174b8881e9a47f424fcf8> 9 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:15] [Debug]    Replacing destination table entry for <d7e6c4222f0174b8881e9a47f424fcf8> with new announce, since it was more recently emitted
[2026-04-13 17:46:15] [Debug]    Destination <d7e6c4222f0174b8881e9a47f424fcf8> is now 9 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:15] [Extra]    Valid announce for <e4fac75fff5af6a1c405f0eb978e47c7> 7 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:15] [Debug]    Destination <e4fac75fff5af6a1c405f0eb978e47c7> is now 7 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:16] [Extra]    Valid announce for <5033840275f77bdacc41a19b42aba77e> 7 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:16] [Debug]    Destination <5033840275f77bdacc41a19b42aba77e> is now 7 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:16] [Extra]    Valid announce for <9275c702df6c8b478dbcec7ea0bf0b08> 6 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:16] [Debug]    Destination <9275c702df6c8b478dbcec7ea0bf0b08> is now 6 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:16] [Extra]    Valid announce for <ab2f4fb3c71cb30bfc9bbaa2738e1b00> 7 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:16] [Debug]    Destination <ab2f4fb3c71cb30bfc9bbaa2738e1b00> is now 7 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:16] [Extra]    Valid announce for <a242915187d12f5b6d5072165fba4b5d> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:16] [Debug]    Replacing destination table entry for <a242915187d12f5b6d5072165fba4b5d> with new announce, since it was more recently emitted
[2026-04-13 17:46:16] [Debug]    Destination <a242915187d12f5b6d5072165fba4b5d> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:17] [Extra]    Valid announce for <a242915187d12f5b6d5072165fba4b5d> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:17] [Extra]    Valid announce for <847512a381cc8022de12012ee575250f> 7 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:17] [Debug]    Destination <847512a381cc8022de12012ee575250f> is now 7 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:17] [Extra]    Valid announce for <b2187b4af3c4cba81b00fcf8bc7897de> 9 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:17] [Debug]    Destination <b2187b4af3c4cba81b00fcf8bc7897de> is now 9 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:17] [Extra]    Valid announce for <95ca807f05d258f7723d5f1f75c29159> 3 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:17] [Debug]    Destination <95ca807f05d258f7723d5f1f75c29159> is now 3 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:17] [Extra]    Valid announce for <95ca807f05d258f7723d5f1f75c29159> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:18] [Extra]    Valid announce for <ffc8d3472451090677fd446837e384ff> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:18] [Debug]    Destination <ffc8d3472451090677fd446837e384ff> is now 5 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:18] [Extra]    Valid announce for <ffc8d3472451090677fd446837e384ff> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:46:18] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:46:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-13 17:46:18] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:46:18] [Warning]  The socket for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] was closed, attempting to reconnect...
[2026-04-13 17:46:18] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-13 17:46:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:46:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:46:18] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-13 17:46:18] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-13 17:46:18] [Warning]  The socket for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] was closed, attempting to reconnect...
[2026-04-13 17:46:18] [Error]    No interfaces could process the outbound packet
[2026-04-13 17:46:19] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:19] [Debug]    Destination <192eed7af8e3311445372f2a43cb63ec> is now 2 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:19] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:19] [Extra]    Valid announce for <192eed7af8e3311445372f2a43cb63ec> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:20] [Extra]    Valid announce for <847512a381cc8022de12012ee575250f> 9 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:20] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:20] [Debug]    Destination <e345f6220682e127cab52c3387436778> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:20] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 5 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:20] [Debug]    Replacing destination table entry for <02aaf088472435718061211d3752c8ed> with new announce, since it was more recently emitted
[2026-04-13 17:46:20] [Debug]    Destination <02aaf088472435718061211d3752c8ed> is now 5 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:20] [Extra]    Valid announce for <e345f6220682e127cab52c3387436778> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:20] [Extra]    Valid announce for <02aaf088472435718061211d3752c8ed> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:21] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:21] [Debug]    Destination <794884194914d03c4e199d9c1f090b0c> is now 6 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:21] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:21] [Debug]    Destination <2d8a25919ea488ce008d3635d9b104c7> is now 4 hops away via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:21] [Extra]    Valid announce for <2fb8b45523a7aed73e80973854aaa9ec> 8 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:21] [Extra]    Valid announce for <794884194914d03c4e199d9c1f090b0c> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:21] [Extra]    Valid announce for <2d8a25919ea488ce008d3635d9b104c7> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:22] [Extra]    Valid announce for <219a60c23a74cf1ede2ee1c56dc790d7> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:22] [Debug]    Destination <219a60c23a74cf1ede2ee1c56dc790d7> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:23] [Extra]    Valid announce for <219a60c23a74cf1ede2ee1c56dc790d7> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-13 17:46:23] [Extra]    Valid announce for <219a60c23a74cf1ede2ee1c56dc790d7> 4 hops away, received via <56180fa4ceca6cd223d60148c01eb8c8> on TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-13 17:46:23] [Debug]    Detaching interfaces
[2026-04-13 17:46:23] [Extra]    Detaching TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-13 17:46:23] [Extra]    Detaching TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-13 17:46:23] [Warning]  An interface error occurred for TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242], the contained exception was: [Errno 9] Bad file descriptor
[2026-04-13 17:46:23] [Warning]  Attempting to reconnect...
