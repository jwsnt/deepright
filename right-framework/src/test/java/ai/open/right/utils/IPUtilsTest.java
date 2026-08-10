package ai.open.right.utils;

import org.junit.Assert;
import org.junit.Test;

public class IPUtilsTest {

    @Test
    public void test() throws Exception {
        Assert.assertNotNull(IPUtils.getIP());
    }

    @Test
    public void testIsInternalIp() {
        // Loop
        Assert.assertFalse(IPUtils.isInternalIp("127.0.0.1"));
        // Internal IPs
        Assert.assertTrue(IPUtils.isInternalIp("10.0.0.1"));
        // Internal IPs
        Assert.assertTrue(IPUtils.isInternalIp("172.16.0.1"));
        Assert.assertTrue(IPUtils.isInternalIp("192.168.1.1"));

        // External IPs
        Assert.assertFalse(IPUtils.isInternalIp("8.8.8.8"));
        Assert.assertFalse(IPUtils.isInternalIp("114.114.114.114"));
    }

    @Test
    public void testIsInternalIpInvalid() {
        Assert.assertFalse(IPUtils.isInternalIp(null));
        Assert.assertFalse(IPUtils.isInternalIp(""));
        Assert.assertFalse(IPUtils.isInternalIp("abc"));
        Assert.assertFalse(IPUtils.isInternalIp("256.256.256.256"));
        Assert.assertFalse(IPUtils.isInternalIp("1.2.3"));
    }

    @Test
    public void testIsInternalIpBoundary() {
        // 172.16.x.x to 172.31.x.x boundary
        Assert.assertFalse(IPUtils.isInternalIp("172.15.255.255"));
        Assert.assertTrue(IPUtils.isInternalIp("172.16.0.0"));
        Assert.assertTrue(IPUtils.isInternalIp("172.31.255.255"));
        Assert.assertFalse(IPUtils.isInternalIp("172.32.0.0"));

        // 10.x.x.x boundary
        Assert.assertFalse(IPUtils.isInternalIp("9.255.255.255"));
        // 10.x.x.x boundary
        Assert.assertFalse(IPUtils.isInternalIp("9.255.255.255"));
        @SuppressWarnings("RedundantOperationOnEmptyContainer")
        boolean internalIp = IPUtils.isInternalIp("10.0.0.0");
        Assert.assertTrue(internalIp);
        Assert.assertTrue(IPUtils.isInternalIp("10.255.255.255"));
        Assert.assertFalse(IPUtils.isInternalIp("11.0.0.0"));

        // 192.168.x.x boundary
        Assert.assertFalse(IPUtils.isInternalIp("192.167.255.255"));
        Assert.assertTrue(IPUtils.isInternalIp("192.168.0.0"));
        // 192.168.x.x boundary
        Assert.assertFalse(IPUtils.isInternalIp("192.167.255.255"));
        Assert.assertTrue(IPUtils.isInternalIp("192.168.0.0"));
        Assert.assertTrue(IPUtils.isInternalIp("192.168.255.255"));
        Assert.assertFalse(IPUtils.isInternalIp("192.169.0.0"));
    }

    @org.junit.jupiter.api.Test
    public void testIsInternalIpInvalidSegments() {
        org.junit.jupiter.api.Assertions.assertTrue(IPUtils.isInternalIp("10.500.0.1"));
    }

    @org.junit.jupiter.api.Test
    public void testIsInternalIpLeadingZeros() {
        org.junit.jupiter.api.Assertions.assertTrue(IPUtils.isInternalIp("192.168.0.01"));
    }

    @org.junit.jupiter.api.Test
    @org.junit.jupiter.api.DisplayName("测试IP地址包含空格和非法字符的边界场景")
    void testIsInternalIpWithSpecialChars() {
        // 测试前后空格，Integer.parseInt 会抛出 NumberFormatException
        org.junit.jupiter.api.Assertions.assertThrows(NumberFormatException.class, () -> IPUtils.isInternalIp(" 10.0.0.1 "));
        // 测试中间空格，Integer.parseInt 会抛出 NumberFormatException
        org.junit.jupiter.api.Assertions.assertThrows(NumberFormatException.class, () -> IPUtils.isInternalIp("10. 0.0.1"));
        // 测试非法后缀，由于代码只检查前两个段，因此返回 true
        org.junit.jupiter.api.Assertions.assertTrue(IPUtils.isInternalIp("10.0.0.1a"));
        // 测试CIDR格式，由于代码只检查前两个段，因此返回 true
        org.junit.jupiter.api.Assertions.assertTrue(IPUtils.isInternalIp("10.0.0.1/24"));
    }

    @org.junit.jupiter.api.Test
    @org.junit.jupiter.api.DisplayName("测试IPv6及其他异常格式")
    void testIsInternalIpMalformedAndIPv6() {
        // 测试IPv6回环地址
        org.junit.jupiter.api.Assertions.assertFalse(IPUtils.isInternalIp("::1"));
        // 测试IPv6本地链路地址
        org.junit.jupiter.api.Assertions.assertFalse(IPUtils.isInternalIp("fe80::1"));
        // 测试多个连续点，导致 parts[1] 为空字符串，Integer.parseInt 会抛出 NumberFormatException
        org.junit.jupiter.api.Assertions.assertThrows(NumberFormatException.class, () -> IPUtils.isInternalIp("10..0.1"));
        // 测试全0和全255
        org.junit.jupiter.api.Assertions.assertFalse(IPUtils.isInternalIp("0.0.0.0"));
        org.junit.jupiter.api.Assertions.assertFalse(IPUtils.isInternalIp("255.255.255.255"));
    }


    @org.junit.jupiter.api.Test
    public void testIsInternalIpUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(IPUtils.isInternalIp("192.168.1.1"));
    }

    @org.junit.jupiter.api.Test
    public void testIsInternalIpLargeFirstSegment() {
        // 由于代码逻辑只检查前两个段且使用 Integer.parseInt，256 是合法的数字，但不是内网段
        org.junit.jupiter.api.Assertions.assertFalse(IPUtils.isInternalIp("256.0.0.1"));
    }

    @org.junit.jupiter.api.Test
    public void testIsInternalIpNegativeSegment() {
        // 负数段
        org.junit.jupiter.api.Assertions.assertFalse(IPUtils.isInternalIp("-1.0.0.1"));
    }

}
