package ai.open.right.utils;

import org.apache.commons.lang3.StringUtils;
import org.springframework.util.CollectionUtils;

import java.net.Inet4Address;
import java.net.InetAddress;
import java.net.NetworkInterface;
import java.util.ArrayList;
import java.util.Enumeration;
import java.util.List;

public class IPUtils {

    public static String getIP() throws Exception {
        List<String> ips = new ArrayList<String>();
        Enumeration<NetworkInterface> interfaces = NetworkInterface.getNetworkInterfaces();
        while (interfaces.hasMoreElements()) {
            NetworkInterface iface = interfaces.nextElement();
            // 排除回环接口和utun类型隧道接口
            if (iface.isLoopback() || iface.getDisplayName().startsWith("utun")) {
                continue;
            }
            Enumeration<InetAddress> addresses = iface.getInetAddresses();
            while (addresses.hasMoreElements()) {
                InetAddress addr = addresses.nextElement();
                if (addr instanceof Inet4Address) {
                    String ip = addr.getHostAddress();
                    if (IPUtils.isInternalIp(ip) || !iface.isVirtual()) {
                        ips.add(ip);
                    }
                }
            }
        }
        return !CollectionUtils.isEmpty(ips) ? ips.getFirst() : null;
    }

    public static boolean isInternalIp(String ip) {
        if (!StringUtils.isEmpty(ip)) {
            String[] parts = ip.split("\\.");
            if (parts.length != 4) {
                return false;
            }
            int first = Integer.parseInt(parts[0]);
            int second = Integer.parseInt(parts[1]);
            return first == 10 || (first == 172 && second >= 16 && second <= 31) || (first == 192 && second == 168);
        } else
            return false;
    }
}
    