package ai.open.right.utils;

import org.apache.commons.lang3.StringUtils;

import java.net.InetAddress;

public class HostUtils {

    public static String getHostName() throws Exception {
        String name = InetAddress.getLocalHost().getHostName();
        return StringUtils.defaultIfEmpty(StringUtils.trim(name), "");
    }
}
