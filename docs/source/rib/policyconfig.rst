.. FlexSwitchL3 documentation master file, created by
   sphinx-quickstart on Mon May 16 11:13:19 2016.
   You can adapt this file completely to your liking, but it should at least
   contain the root `toctree` directive.

.. sectnum::

Policy Objects Configuration Examples 
========================================

Configuring Policy Condition with Rest API
------------------------------------------

**COMMAND:**
::

     curl -H "Content-Type: application/json" -d '{"Name":"Match40", "ConditionType":"MatchDstIpPrefix", "IpPrefix":"40.1.3.0/24", "MaskLengthRange":"exact"}' http://localhost:8080/public/v1/config/PolicyCondition

	

**OPTIONS:**

+-----------------+-------------+----------------------------------------------+----------+----------+
| Variables       | Type        |  Description                                 | Required |  Default |   
+=================+=============+==============================================+==========+==========+ 
|Name             | string      | Name of the policy condition                 |   Yes    |    N/A   |          
+-----------------+-------------+----------------------------------------------+----------+----------+
|ConditionType    | string      | Condition type                         .     |   Yes    |    N/A   |
+-----------------+-------------+----------------------------------------------+----------+----------+
|                 |             |    MatchProtocol                             |          | N/A      |
+-----------------+-------------+----------------------------------------------+----------+----------+
|                 |             |    MatchDstIpPrefix                          |          | N/A      |
+-----------------+-------------+----------------------------------------------+----------+----------+
|                 |             | Protocol to match on when the condition      |          |          |
| Protocol        | String      | type is MatchProtocol                        |    No    |          |
+-----------------+-------------+----------------------------------------------+----------+----------+ 
|                 |             | Prefix to match on when the condition        |          |          |
| IpPrefix        | String      | type is MatchDstIpPrefix/MatchSrcIpPrefix    |    No    |          |
+-----------------+-------------+----------------------------------------------+----------+----------+ 
|                 |             | Used along with IpPrefix and specifies       |          |          |
| MaskLengthRange | String      | whether to match the exact prefix or a range |    No    |          |
+-----------------+-------------+----------------------------------------------+----------+----------+ 

**EXAMPLE:**
::
	
      //condition to match on a network
      curl -H "Content-Type: application/json" -d '{"Name":"Match40", "ConditionType":"MatchDstIpPrefix", "IpPrefix":"40.1.3.0/24", "MaskLengthRange":"exact"}' http://localhost:8080/public/v1/config/PolicyCondition
      {"ObjectId":"a8c98e74-eeb7-4580-6ff9-23b3b48d76f9","Error":""}

      curl -X GET --header 'Content-Type: application/json' --header 'Accept: application/json' http://localhost:8080/public/v1/config/PolicyConditions | python -m json.tool
      % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
      100   257  100   257    0     0  42138      0 --:--:-- --:--:-- --:--:-- 51400
      {
          "CurrentMarker": 0,
           "MoreExist": false,
           "NextMarker": 0,
           "ObjCount": 1,
           "Objects": [
               {
                   "Object": {
                       "ConditionType": "MatchDstIpPrefix",
                       "IpPrefix": "40.1.3.0/24",
                       "MaskLengthRange": "exact",
                       "Name": "Match40",
                       "Protocol": ""
                   },
                   "ObjectId": "a8c98e74-eeb7-4580-6ff9-23b3b48d76f9"
               }
           ]
       }
	
Configuring Policy Statement with Rest API
------------------------------------------

**COMMAND:**
::

    curl -H "Content-Type: application/json" -d '{"Name":"Stmt1", "MatchConditions":"all", "Conditions":["Match40"], "Action":"permit"}' http://localhost:8080/public/v1/config/PolicyStmt


**OPTIONS:**

+----------------+-------------+------------------------------------------------+----------+----------+
| Variables      | Type        |  Description                                   | Required |  Default |   
+================+=============+================================================+==========+==========+ 
|Name            | string      | Name of the policy Statement                   |   Yes    |    N/A   |          
+----------------+-------------+------------------------------------------------+----------+----------+
|MatchConditions | string      | Specifies whether to match all or any          |   Yes    |    "all" |
|                |             | of the conditions                         .    |          |          |
+----------------+-------------+------------------------------------------------+----------+----------+
| Conditions     | []String    | list of conditions used to evaluate this       |    Yes   |          |
|                |             | statement                         .            |          |          |
+----------------+-------------+------------------------------------------------+----------+----------+ 
| Action         | String      | Action on a successful evaluation of statement |    No    |    "deny"|
+----------------+-------------+------------------------------------------------+----------+----------+ 


**EXAMPLE:**
::
	
      curl -H "Content-Type: application/json" -d '{"Name":"Stmt1", "MatchConditions":"all", "Conditions":["Match40"], "Action":"permit"}' http://localhost:8080/public/v1/config/PolicyStmt
      {"ObjectId":"e73825a0-b307-4498-76ad-5c8552c866e3","Error":""}
	
      curl -X GET --header 'Content-Type: application/json' --header 'Accept: application/json' http://localhost:8080/public/v1/config/PolicyStmts | python -m json.tool
      % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
      100   222  100   222    0     0  66486      0 --:--:-- --:--:-- --:--:-- 74000
      {
          "CurrentMarker": 0,
          "MoreExist": false,
          "NextMarker": 0,
          "ObjCount": 1,
          "Objects": [
              {
                "Object": {
                    "Action": "permit",
                    "Conditions": [
                        "Match40"
                    ],
                    "MatchConditions": "all",
                    "Name": "Stmt1"
                },
                "ObjectId": "e73825a0-b307-4498-76ad-5c8552c866e3"
             }
        ]
    }

	
Configuring Policy Definition with Rest API
-------------------------------------------

**COMMAND:**
::
	
     curl -H "Content-Type: application/json" -d '{"Name":"Policy1", "Priority":1,"MatchType":"all", "PolicyType":"ALL", "StatementList":[{"Priority":1,"Statement":"Stmt1"}]}' http://localhost:8080/public/v1/config/PolicyDefinition

	

**OPTIONS:**

+--------------+--------------------------------+---------------------------------------------+----------+----------+
| Variables    | Type                           |  Description                                | Required |  Default |   
+==============+================================+=============================================+==========+==========+ 
|Name          | string                         | Name of the policy definition               |   Yes    |    N/A   |          
+--------------+--------------------------------+---------------------------------------------+----------+----------+
|Priority      | int32                          | Priority of the policy                      |   Yes    |    N/A   |
+--------------+--------------------------------+---------------------------------------------+----------+----------+
|MatchType     | string                         | Specifies if any or all of the statements   |   No     |    "all" |
|              |                                | of the policy needs to be executed          |          |          |
+--------------+--------------------------------+---------------------------------------------+----------+----------+
|              |                                | List of {priority,statement}s specifies     |          |          |
| StatementList| []PolicyDefinitionStmtPriority | statements in the policy and their priority |    No    |          |
+--------------+--------------------------------+---------------------------------------------+----------+----------+ 


**EXAMPLE:**
::
	
      curl -H "Content-Type: application/json" -d '{"Name":"Policy1", "Priority":1,"MatchType":"all", "PolicyType":"ALL", "StatementList":[{"Priority":1,"Statement":"Stmt1"}]}' http://localhost:8080/public/v1/config/PolicyDefinition
      {"ObjectId":"770dde8f-b357-4c38-4e25-8289e507614a","Error":""}

      curl -X GET --header 'Content-Type: application/json' --header 'Accept: application/json' http://localhost:8080/public/v1/config/PolicyDefinitions | python -m json.tool
       % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
       100   260  100   260    0     0  61743      0 --:--:-- --:--:-- --:--:-- 65000
       {
           "CurrentMarker": 0,
           "MoreExist": false,
           "NextMarker": 0,
           "ObjCount": 1,
           "Objects": [
               {
                   "Object": {
                       "MatchType": "all",
                       "Name": "Policy1",
                       "PolicyType": "ALL",
                       "Priority": 1,
                       "StatementList": [
                       {
                           "Priority": 1,
                           "Statement": "Stmt1"
                       }
                   ]
               },
               "ObjectId": "770dde8f-b357-4c38-4e25-8289e507614a"
            }
        ]
    }

