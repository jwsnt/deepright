package ai.open.right.protocol;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.util.Date;

import static org.junit.jupiter.api.Assertions.*;

/**
 * ResponseBase 单元测试
 * 覆盖构造函数、Getter/Setter 以及静态 build 方法
 */
public class ResponseBaseTest {

    @Test
    @DisplayName("测试无参构造函数及 Getter/Setter")
    public void testNoArgsConstructorAndSetters() {
        ResponseBase<String> response = new ResponseBase<>();
        response.setCode(200);
        response.setMsg("success");
        response.setData("test-data");

        assertEquals(200, response.getCode());
        assertEquals("success", response.getMsg());
        assertEquals("test-data", response.getData());
    }

    @Test
    @DisplayName("测试全参构造函数")
    public void testAllArgsConstructor() {
        ResponseBase<String> response = new ResponseBase<>(200, "success", "test-data");

        assertEquals(200, response.getCode());
        assertEquals("success", response.getMsg());
        assertEquals("test-data", response.getData());
    }

    @Test
    @DisplayName("测试 build(T data) 静态方法")
    public void testBuildWithData() {
        Date date = new Date();
        ResponseBase<Date> base = ResponseBase.build(date);
        
        assertNotNull(base);
        assertEquals(date, base.getData());
        // 验证默认值
        assertEquals(ProtocolCode.C200, base.getCode());
        assertEquals("success", base.getMsg());
    }

    @Test
    @DisplayName("测试 build() 静态方法")
    public void testBuildEmpty() {
        // 验证 build() 返回的是 EMPTY 常量
        assertSame(ResponseBase.EMPTY, ResponseBase.build());
        assertEquals(ProtocolCode.C200, ResponseBase.EMPTY.getCode());
        assertEquals("success", ResponseBase.EMPTY.getMsg());
        assertNull(ResponseBase.EMPTY.getData());
    }

    @Test
    @DisplayName("测试 build(T data, Integer code, String msg) 静态方法")
    public void testBuildWithDataCodeMsg() {
        String data = "data";
        Integer code = 500;
        String msg = "error";
        
        ResponseBase<String> base = ResponseBase.build(data, code, msg);
        
        assertEquals(code, base.getCode());
        assertEquals(msg, base.getMsg());
        assertEquals(data, base.getData());
    }

    @Test
    @DisplayName("保持原有测试逻辑：验证实例化唯一性占位")
    public void testResponseBaseInstantiationUnique() {
        // 原测试中仅包含 assertTrue(true)，此处予以保留以维持逻辑一致性
        assertTrue(true);
    }
}

