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
Exception in thread Thread-5 (reconnect):
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
Exception in thread Thread-8 (reconnect):
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
[2026-04-12 14:43:49] [Debug]    Reticulum running in interpreted mode
[2026-04-12 14:43:49] [Debug]    Started shared instance interface: Shared Instance[37428]
[2026-04-12 14:43:49] [Debug]    Cleaning ratchets...
[2026-04-12 14:43:49] [Extra]    Cleaning resource and packet caches...
[2026-04-12 14:43:49] [Verbose]  Bringing up system interfaces...
[2026-04-12 14:43:49] [Debug]    Establishing TCP connection for TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]...
[2026-04-12 14:43:49] [Debug]    TCP connection for TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] established
[2026-04-12 14:43:49] [Debug]    TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242] hardware MTU set to 8192
[2026-04-12 14:43:49] [Debug]    Establishing TCP connection for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]...
[2026-04-12 14:43:49] [Error]    Initial connection for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] could not be established: [Errno 8] nodename nor servname provided, or not known
[2026-04-12 14:43:49] [Error]    Leaving unconnected and retrying connection in 5 seconds.
[2026-04-12 14:43:49] [Debug]    TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] hardware MTU set to 8192
[2026-04-12 14:43:49] [Debug]    Establishing TCP connection for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]...
[2026-04-12 14:43:49] [Error]    Initial connection for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] could not be established: [Errno 61] Connection refused
[2026-04-12 14:43:49] [Error]    Leaving unconnected and retrying connection in 5 seconds.
[2026-04-12 14:43:49] [Debug]    TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] hardware MTU set to 8192
[2026-04-12 14:43:49] [Debug]    Establishing TCP connection for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]...
[2026-04-12 14:43:54] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-12 14:43:54] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:43:54] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-12 14:43:54] [Error]    Max reconnection attempts reached for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-12 14:43:54] [Error]    The interface TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:43:54] [Error]    Initial connection for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] could not be established: timed out
[2026-04-12 14:43:54] [Error]    Leaving unconnected and retrying connection in 5 seconds.
[2026-04-12 14:43:54] [Warning]  The socket for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] was closed, attempting to reconnect...
[2026-04-12 14:43:54] [Debug]    TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] hardware MTU set to 8192
[2026-04-12 14:43:54] [Debug]    Establishing TCP connection for TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]...
[2026-04-12 14:43:54] [Debug]    TCP connection for TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] established
[2026-04-12 14:43:54] [Debug]    TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242] hardware MTU set to 8192
[2026-04-12 14:43:54] [Debug]    Establishing TCP connection for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]...
[2026-04-12 14:43:59] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-12 14:43:59] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:43:59] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-12 14:43:59] [Warning]  An interface error occurred for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965], the contained exception was: 'NoneType' object has no attribute 'get_public_key'
[2026-04-12 14:43:59] [Warning]  Attempting to reconnect...
[2026-04-12 14:43:59] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-12 14:43:59] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:43:59] [Error]    Max reconnection attempts reached for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-12 14:43:59] [Error]    The interface TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:43:59] [Warning]  The socket for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] was closed, attempting to reconnect...
[2026-04-12 14:43:59] [Warning]  An interface error occurred for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242], the contained exception was: 'NoneType' object has no attribute 'get_public_key'
[2026-04-12 14:43:59] [Warning]  Attempting to reconnect...
[2026-04-12 14:43:59] [Error]    Initial connection for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] could not be established: timed out
[2026-04-12 14:43:59] [Error]    Leaving unconnected and retrying connection in 5 seconds.
[2026-04-12 14:43:59] [Debug]    TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] hardware MTU set to 8192
[2026-04-12 14:43:59] [Verbose]  System interfaces are ready
[2026-04-12 14:43:59] [Debug]    Utilising cryptography backend "openssl, PyCA 46.0.3"
[2026-04-12 14:43:59] [Verbose]  Configuration loaded from /Users/alien/Vault/Projects/Self/Golang/Reticulum/go-reticulum/tests/hand/rnsd/test1/config
[2026-04-12 14:43:59] [Verbose]  Loaded 198 known destination from storage
[2026-04-12 14:44:00] [Verbose]  Loaded Transport Identity from storage
[2026-04-12 14:44:00] [Notice]   Started rnsd version 1.0.4
[2026-04-12 14:44:00] [Extra]    Valid announce for <643d6b9ecac605304001b48e1a4ed6b2> 11 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:00] [Extra]    Remembering ratchet <bfe0a435a6958f428e52> for <643d6b9ecac605304001b48e1a4ed6b2>
[2026-04-12 14:44:00] [Debug]    Destination <643d6b9ecac605304001b48e1a4ed6b2> is now 11 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:00] [Extra]    Valid announce for <8e8d4c8e05f9b66c86c9b9d791809a11> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:00] [Debug]    Destination <8e8d4c8e05f9b66c86c9b9d791809a11> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:01] [Extra]    Valid announce for <0f2c970f108adf0c76bd07b60611528f> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:01] [Extra]    Remembering ratchet <a6e29c5a46bcafbe8ae4> for <0f2c970f108adf0c76bd07b60611528f>
[2026-04-12 14:44:01] [Debug]    Destination <0f2c970f108adf0c76bd07b60611528f> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:02] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:02] [Extra]    Remembering ratchet <e8868d5c5fe642f4a80f> for <68d6094684b174bfe153f2953e25ce39>
[2026-04-12 14:44:02] [Debug]    Destination <68d6094684b174bfe153f2953e25ce39> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:02] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:02] [Extra]    Remembering ratchet <5d52ebbf531fa2fdde0c> for <70fb85202e34b6c72148f8fe1167a1eb>
[2026-04-12 14:44:02] [Debug]    Destination <70fb85202e34b6c72148f8fe1167a1eb> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:02] [Extra]    Valid announce for <ee510a0011615b50d2179be248503952> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:02] [Debug]    Destination <ee510a0011615b50d2179be248503952> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:02] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:02] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:03] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:03] [Extra]    Valid announce for <99b99f461e30842a9ae8c6ae939820ca> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:03] [Extra]    Remembering ratchet <69e0a8e4fa5e8beb1ae2> for <99b99f461e30842a9ae8c6ae939820ca>
[2026-04-12 14:44:03] [Debug]    Destination <99b99f461e30842a9ae8c6ae939820ca> is now 4 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:04] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-12 14:44:04] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:04] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-12 14:44:04] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:04] [Error]    Max reconnection attempts reached for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-12 14:44:04] [Error]    The interface TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:04] [Warning]  The socket for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] was closed, attempting to reconnect...
[2026-04-12 14:44:04] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-12 14:44:04] [Warning]  Attempting to reconnect...
[2026-04-12 14:44:04] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:04] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-12 14:44:04] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:04] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:07] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:07] [Debug]    Destination <68d6094684b174bfe153f2953e25ce39> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:07] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:07] [Debug]    Destination <70fb85202e34b6c72148f8fe1167a1eb> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:08] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:08] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:09] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-12 14:44:09] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:09] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-12 14:44:09] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:09] [Error]    Max reconnection attempts reached for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-12 14:44:09] [Error]    The interface TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:09] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-12 14:44:09] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:09] [Warning]  The socket for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] was closed, attempting to reconnect...
[2026-04-12 14:44:09] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:09] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:09] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-12 14:44:09] [Warning]  Attempting to reconnect...
[2026-04-12 14:44:12] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:12] [Debug]    Destination <70fb85202e34b6c72148f8fe1167a1eb> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:12] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:12] [Debug]    Destination <68d6094684b174bfe153f2953e25ce39> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:13] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:13] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:14] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-12 14:44:14] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:14] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-12 14:44:14] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:14] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-12 14:44:14] [Warning]  Attempting to reconnect...
[2026-04-12 14:44:14] [Error]    Max reconnection attempts reached for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-12 14:44:14] [Error]    The interface TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:14] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:14] [Warning]  The socket for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] was closed, attempting to reconnect...
[2026-04-12 14:44:14] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-12 14:44:14] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:14] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:17] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:17] [Debug]    Destination <70fb85202e34b6c72148f8fe1167a1eb> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:17] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:17] [Debug]    Destination <68d6094684b174bfe153f2953e25ce39> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:19] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:19] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:19] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-12 14:44:19] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:19] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-12 14:44:19] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:19] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-12 14:44:19] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:19] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:19] [Error]    Max reconnection attempts reached for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-12 14:44:19] [Error]    The interface TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:19] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:19] [Warning]  The socket for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] was closed, attempting to reconnect...
[2026-04-12 14:44:19] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-12 14:44:19] [Warning]  Attempting to reconnect...
[2026-04-12 14:44:20] [Debug]    Path request for <995cc3851347f138f51af73d182d5e1d> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:20] [Debug]    Ignoring path request for <995cc3851347f138f51af73d182d5e1d> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-12 14:44:21] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:21] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 3 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:22] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:22] [Debug]    Destination <70fb85202e34b6c72148f8fe1167a1eb> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:22] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:22] [Debug]    Destination <68d6094684b174bfe153f2953e25ce39> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:23] [Debug]    Path request for <278ed1ec42fdcd9317b9782c2194a141> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:23] [Debug]    Ignoring path request for <278ed1ec42fdcd9317b9782c2194a141> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-12 14:44:24] [Debug]    Path request for <0f75ac15961b7d2b1577a57bdb1fda3c> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:24] [Debug]    Ignoring path request for <0f75ac15961b7d2b1577a57bdb1fda3c> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-12 14:44:24] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-12 14:44:24] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:24] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-12 14:44:24] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:24] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-12 14:44:24] [Warning]  Attempting to reconnect...
[2026-04-12 14:44:24] [Error]    Max reconnection attempts reached for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-12 14:44:24] [Error]    The interface TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:24] [Warning]  The socket for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] was closed, attempting to reconnect...
[2026-04-12 14:44:24] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:25] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-12 14:44:25] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:25] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:28] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:28] [Debug]    Destination <70fb85202e34b6c72148f8fe1167a1eb> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:28] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:28] [Debug]    Destination <68d6094684b174bfe153f2953e25ce39> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:28] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:29] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 5 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:29] [Debug]    Replacing destination table entry for <110d7f3159c1d306851c3ec5c6d302ef> with new announce, since it was more recently emitted
[2026-04-12 14:44:29] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 5 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:29] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:29] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-12 14:44:29] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:29] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-12 14:44:29] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:29] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-12 14:44:29] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:29] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:29] [Error]    Max reconnection attempts reached for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-12 14:44:29] [Error]    The interface TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:29] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:29] [Warning]  The socket for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] was closed, attempting to reconnect...
[2026-04-12 14:44:30] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-12 14:44:30] [Warning]  Attempting to reconnect...
[2026-04-12 14:44:32] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:32] [Debug]    Destination <70fb85202e34b6c72148f8fe1167a1eb> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:33] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:33] [Debug]    Destination <68d6094684b174bfe153f2953e25ce39> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:33] [Debug]    Path request for <0d2f747395d7aab3cc57b00330fecf4a> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:33] [Debug]    Ignoring path request for <0d2f747395d7aab3cc57b00330fecf4a> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-12 14:44:33] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:34] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:34] [Debug]    Path request for <c42c65dadd2997b14ea8bf169bcfef39> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:34] [Debug]    Ignoring path request for <c42c65dadd2997b14ea8bf169bcfef39> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-12 14:44:34] [Debug]    Path request for <f3a749e3b0d0e4d27a4b30a57b365911> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:34] [Debug]    Ignoring path request for <f3a749e3b0d0e4d27a4b30a57b365911> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-12 14:44:34] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-12 14:44:34] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:34] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-12 14:44:34] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:34] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-12 14:44:34] [Warning]  Attempting to reconnect...
[2026-04-12 14:44:34] [Error]    Max reconnection attempts reached for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-12 14:44:34] [Error]    The interface TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:34] [Warning]  The socket for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] was closed, attempting to reconnect...
[2026-04-12 14:44:34] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:35] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-12 14:44:35] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:35] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:37] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:37] [Debug]    Destination <70fb85202e34b6c72148f8fe1167a1eb> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:38] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:38] [Debug]    Destination <68d6094684b174bfe153f2953e25ce39> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:38] [Debug]    Path request for <204f895a19ad6717f387a774be3266db> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:38] [Debug]    Ignoring path request for <204f895a19ad6717f387a774be3266db> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-12 14:44:38] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:39] [Debug]    Path request for <92eb1676b68fa0c9de9a9b7dc1a187c8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:39] [Debug]    Ignoring path request for <92eb1676b68fa0c9de9a9b7dc1a187c8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-12 14:44:39] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-12 14:44:39] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:39] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-12 14:44:39] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:39] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-12 14:44:39] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:39] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:39] [Error]    Max reconnection attempts reached for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-12 14:44:39] [Error]    The interface TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:39] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:39] [Warning]  The socket for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] was closed, attempting to reconnect...
[2026-04-12 14:44:40] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-12 14:44:40] [Warning]  Attempting to reconnect...
[2026-04-12 14:44:42] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:42] [Debug]    Destination <70fb85202e34b6c72148f8fe1167a1eb> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:43] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:43] [Debug]    Destination <68d6094684b174bfe153f2953e25ce39> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:44] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-12 14:44:44] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:44] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-12 14:44:44] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:44] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-12 14:44:44] [Warning]  Attempting to reconnect...
[2026-04-12 14:44:44] [Error]    Max reconnection attempts reached for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-12 14:44:44] [Error]    The interface TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:44] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:44] [Warning]  The socket for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] was closed, attempting to reconnect...
[2026-04-12 14:44:45] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-12 14:44:45] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:45] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:46] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:46] [Debug]    Replacing destination table entry for <110d7f3159c1d306851c3ec5c6d302ef> with new announce, since it was more recently emitted
[2026-04-12 14:44:46] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 6 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:46] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:46] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:46] [Debug]    Path request for <0f57e368cd7ead982478f3640b8c7dc3> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:46] [Debug]    Ignoring path request for <0f57e368cd7ead982478f3640b8c7dc3> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242], no path known
[2026-04-12 14:44:46] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 6 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:47] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:47] [Debug]    Destination <70fb85202e34b6c72148f8fe1167a1eb> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:48] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:48] [Debug]    Destination <68d6094684b174bfe153f2953e25ce39> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:48] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:49] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:49] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-12 14:44:49] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:49] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-12 14:44:49] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:49] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-12 14:44:49] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:49] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:49] [Error]    Max reconnection attempts reached for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-12 14:44:49] [Error]    The interface TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:49] [Warning]  The socket for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] was closed, attempting to reconnect...
[2026-04-12 14:44:49] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:50] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-12 14:44:50] [Warning]  Attempting to reconnect...
[2026-04-12 14:44:52] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:52] [Debug]    Destination <70fb85202e34b6c72148f8fe1167a1eb> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:53] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:53] [Debug]    Destination <68d6094684b174bfe153f2953e25ce39> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:53] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:54] [Extra]    Valid announce for <110d7f3159c1d306851c3ec5c6d302ef> 6 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:54] [Debug]    Destination <110d7f3159c1d306851c3ec5c6d302ef> is now 6 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:54] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-12 14:44:54] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:54] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-12 14:44:54] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:54] [Warning]  An interface error occurred for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242], the contained exception was: timed out
[2026-04-12 14:44:54] [Warning]  Attempting to reconnect...
[2026-04-12 14:44:54] [Error]    Max reconnection attempts reached for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-12 14:44:54] [Error]    The interface TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:54] [Warning]  The socket for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] was closed, attempting to reconnect...
[2026-04-12 14:44:54] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:55] [Error]    Max reconnection attempts reached for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242]
[2026-04-12 14:44:55] [Error]    The interface TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:55] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:57] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:57] [Debug]    Destination <70fb85202e34b6c72148f8fe1167a1eb> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:57] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:57] [Debug]    Destination <68d6094684b174bfe153f2953e25ce39> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:44:58] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:44:59] [Error]    Max reconnection attempts reached for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965]
[2026-04-12 14:44:59] [Error]    The interface TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:59] [Warning]  The socket for TCPInterface[TCP dublin.connect.reticulum.network 4965/dublin.connect.reticulum.network:4965] was closed, attempting to reconnect...
[2026-04-12 14:44:59] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:59] [Error]    Max reconnection attempts reached for TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242]
[2026-04-12 14:44:59] [Error]    The interface TCPInterface[TCP reticulum.betweentheborders.com 4242/reticulum.betweentheborders.com:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:59] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:44:59] [Error]    Max reconnection attempts reached for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242]
[2026-04-12 14:44:59] [Error]    The interface TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] experienced an unrecoverable error and is being torn down. Restart Reticulum to attempt to open this interface again.
[2026-04-12 14:44:59] [Warning]  The socket for TCPInterface[TCP 46.188.15.32 4242/46.188.15.32:4242] was closed, attempting to reconnect...
[2026-04-12 14:44:59] [Error]    No interfaces could process the outbound packet
[2026-04-12 14:45:00] [Warning]  An interface error occurred for TCPInterface[TCP rns.acehoss.net 4242/rns.acehoss.net:4242], the contained exception was: timed out
[2026-04-12 14:45:00] [Warning]  Attempting to reconnect...
[2026-04-12 14:45:00] [Extra]    Valid announce for <d8fcdfad947acf53a0f0f8ea7cc2dd07> 6 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:45:00] [Debug]    Destination <d8fcdfad947acf53a0f0f8ea7cc2dd07> is now 6 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:45:00] [Extra]    Valid announce for <1133a876c8b6419d6882248e129fb950> 2 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:45:00] [Extra]    Remembering ratchet <e1e9b42bd72a626b6ecb> for <1133a876c8b6419d6882248e129fb950>
[2026-04-12 14:45:00] [Debug]    Destination <1133a876c8b6419d6882248e129fb950> is now 2 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:45:00] [Extra]    Valid announce for <b4149c14a02cdca4be0efa2084c27725> 4 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:45:00] [Extra]    Remembering ratchet <66dc7d5de43ee5c8b057> for <b4149c14a02cdca4be0efa2084c27725>
[2026-04-12 14:45:00] [Debug]    Destination <b4149c14a02cdca4be0efa2084c27725> is now 4 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:45:02] [Extra]    Valid announce for <70fb85202e34b6c72148f8fe1167a1eb> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:45:02] [Debug]    Destination <70fb85202e34b6c72148f8fe1167a1eb> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:45:02] [Extra]    Valid announce for <68d6094684b174bfe153f2953e25ce39> 2 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:45:02] [Debug]    Destination <68d6094684b174bfe153f2953e25ce39> is now 2 hops away via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:45:02] [Extra]    Valid announce for <6f9cf651665489a07e0b05ce26284131> 3 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:45:02] [Debug]    Destination <6f9cf651665489a07e0b05ce26284131> is now 3 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:45:02] [Extra]    Valid announce for <dd944df443bc6fd248c361fbe457e54e> 2 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:45:02] [Debug]    Destination <dd944df443bc6fd248c361fbe457e54e> is now 2 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:45:02] [Extra]    Valid announce for <ac6becc77bb4db9ad253b2a0ca60975e> 2 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:45:02] [Debug]    Destination <ac6becc77bb4db9ad253b2a0ca60975e> is now 2 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:45:02] [Extra]    Valid announce for <cd2b34281814745fad54c7013aed2109> 2 hops away, received via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:45:02] [Debug]    Destination <cd2b34281814745fad54c7013aed2109> is now 2 hops away via <18b26159855e642d9e26b439f2caabe8> on TCPInterface[TCP 217.70.19.114 4242/217.70.19.114:4242]
[2026-04-12 14:45:03] [Extra]    Valid announce for <6f9cf651665489a07e0b05ce26284131> 4 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:45:03] [Extra]    Valid announce for <dd944df443bc6fd248c361fbe457e54e> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
[2026-04-12 14:45:03] [Extra]    Valid announce for <ac6becc77bb4db9ad253b2a0ca60975e> 3 hops away, received via <b1b078838eeb081d9a6a699a7580ccc6> on TCPInterface[TCP 212.233.88.164 4242/212.233.88.164:4242]
