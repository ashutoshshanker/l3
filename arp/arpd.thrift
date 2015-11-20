namespace go arpd
typedef i32 int

service ARPService
{
    int RestolveArpIPV4(1:string destNetIp, 2:string ifName);
}
