.. FlexSwitchL3 documentation master file, created by
   sphinx-quickstart on Mon May 16 11:13:19 2016.
   You can adapt this file completely to your liking, but it should at least
   contain the root `toctree` directive.

.. sectnum::

Configuration Examples 
========================================

Configuring Route with Rest API
-------------------------------

**COMMAND:**
::
	
     curl -H "Content-Type: application/json" -d '{"DestinationNw": "40.1.10.0", "NetworkMask": "255.255.255.0", "Protocol": "STATIC", "NextHop": [{"NextHopIp": "40.1.1.2", "NextHopIntRef":"lo1"}]}' http://localhost:8080/public/v1/config/IPv4Route
	

**OPTIONS:**

+--------------+-------------+-------------------------------------------+----------+----------+
| Variables    | Type        |  Description                              | Required |  Default |   
+==============+=============+===========================================+==========+==========+ 
|DestinationNw | string      | IP address of the destination nw          |   Yes    |    N/A   |          
+--------------+-------------+-------------------------------------------+----------+----------+
|Protocol      | string      | Route Protocol type                    .  |    No    | "STATIC" |
+--------------+-------------+-------------------------------------------+----------+----------+
| NextHop      | NextHopInfo | Next hop details :                        |    Yes   |          |
+--------------+-------------+-------------------------------------------+----------+----------+
|              | string      |    Next Hop IP                            |    Yes   | N/A      |
+--------------+-------------+-------------------------------------------+----------+----------+
|              | string      |    Next Hop Interface                     |    Yes   | N/A      |
+--------------+-------------+-------------------------------------------+----------+----------+
|              | int32       |    Weight of the next hop  (0..31)        |    No    |   0      |
+--------------+-------------+-------------------------------------------+----------+----------+ 
| NullRoute    | Boolean     | Specify if this is Null Route             |    No    |  false   |
+--------------+-------------+-------------------------------------------+----------+----------+ 
| Cost         | int32       | Specify the cost of the Route             |    No    |  0       |
+--------------+-------------+-------------------------------------------+----------+----------+ 


**EXAMPLE:**
::
	
	 curl -H "Content-Type: application/json" -d '{"DestinationNw": "40.1.10.0", "NetworkMask": "255.255.255.0", "Protocol": "STATIC", "NextHop": [{"NextHopIp": "40.1.1.2", "NextHopIntRef":"lo1"}]}' http://localhost:8080/public/v1/config/IPv4Route
     {"ObjectId":"99181161-6438-40df-7926-a0dd78d5dd29","Error":""}

      curl -X GET --header 'Content-Type: application/json' --header 'Accept: application/json' http://localhost:8080/public/v1/config/IPv4Routes | python -m json.tool
      % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
     100   315  100   315    0     0  76382      0 --:--:-- --:--:-- --:--:--  102k
     {
        "CurrentMarker": 0,
        "MoreExist": false,
        "NextMarker": 0,
        "ObjCount": 1,
        "Objects": [
            {
                "Object": {
                    "Cost": 0,
                    "DestinationNw": "40.1.10.0",
                    "NetworkMask": "255.255.255.0",
                    "NextHop": [
                        {
                            "NextHopIntRef": "lo1",
                            "NextHopIp": "40.1.1.2",
                            "Weight": 0
                        }
                    ],
                    "NullRoute": false,
                    "Protocol": "STATIC"
                },
                "ObjectId": "99181161-6438-40df-7926-a0dd78d5dd29"
            }
        ]
    }
	
Configuring ECMP Route with Rest API
------------------------------------

**COMMAND:**
::

    First route 
    curl -H "Content-Type: application/json" -d '{"DestinationNw": "40.1.10.0", "NetworkMask": "255.255.255.0", "Protocol": "STATIC", "NextHop": [{"NextHopIp": "40.1.1.2", "NextHopIntRef":"lo1"}]}' http://localhost:8080/public/v1/config/IPv4Route
	
	Subsequent next hops add:
	curl -X PATCH -H "Content-Type: application/json" -d '{"op":"add", "DestinationNw": "40.1.10.0", "NetworkMask": "255.255.255.0", "NextHop": [{"NextHopIp": "40.1.2.2", "NextHopIntRef":"lo2"}]}' http://localhost:8080/public/v1/config/IPv4Route



**EXAMPLE:**
::
	
	 curl -H "Content-Type: application/json" -d '{"DestinationNw": "40.1.10.0", "NetworkMask": "255.255.255.0", "Protocol": "STATIC", "NextHop": [{"NextHopIp": "40.1.1.2", "NextHopIntRef":"lo1"}]}' http://localhost:8080/public/v1/config/IPv4Route
     {"ObjectId":"99181161-6438-40df-7926-a0dd78d5dd29","Error":""}

      curl -X GET --header 'Content-Type: application/json' --header 'Accept: application/json' http://localhost:8080/public/v1/config/IPv4Routes | python -m json.tool
      % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
     100   315  100   315    0     0  76382      0 --:--:-- --:--:-- --:--:--  102k
     {
        "CurrentMarker": 0,
        "MoreExist": false,
        "NextMarker": 0,
        "ObjCount": 1,
        "Objects": [
            {
                "Object": {
                    "Cost": 0,
                    "DestinationNw": "40.1.10.0",
                    "NetworkMask": "255.255.255.0",
                    "NextHop": [
                        {
                            "NextHopIntRef": "lo1",
                            "NextHopIp": "40.1.1.2",
                            "Weight": 0
                        }
                    ],
                    "NullRoute": false,
                    "Protocol": "STATIC"
                },
                "ObjectId": "99181161-6438-40df-7926-a0dd78d5dd29"
            }
        ]
    }

	curl -X PATCH -H "Content-Type: application/json" -d '{"op":"add", "DestinationNw": "40.1.10.0", "NetworkMask": "255.255.255.0", "NextHop": [{"NextHopIp": "40.1.2.2", "NextHopIntRef":"lo2"}]}' http://localhost:8080/public/v1/config/IPv4Route

    display after the second next hop add:
	
	# curl -X GET --header 'Content-Type: application/json' --header 'Accept: application/json' http://localhost:8080/public/v1/config/IPv4Routes | python -m json.tool  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
    100   373  100   373    0     0   103k      0 --:--:-- --:--:-- --:--:--  121k
    {
        "CurrentMarker": 0,
        "MoreExist": false,
        "NextMarker": 0,
        "ObjCount": 1,
        "Objects": [
           {
                "Object": {
                    "Cost": 0,
                    "DestinationNw": "40.1.10.0",
                    "NetworkMask": "255.255.255.0",
                    "NextHop": [
                        {
                            "NextHopIntRef": "lo2",
                            "NextHopIp": "40.1.2.2",
                            "Weight": 0
                        },
                        {
                            "NextHopIntRef": "lo1",
                            "NextHopIp": "40.1.1.2",
                            "Weight": 0
                        }
                    ],
                    "NullRoute": false,
                    "Protocol": "STATIC"
                },
                "ObjectId": "99181161-6438-40df-7926-a0dd78d5dd29"
            }
        ]
    }
	


