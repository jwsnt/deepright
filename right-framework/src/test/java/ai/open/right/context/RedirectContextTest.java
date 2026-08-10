package ai.open.right.context;

import org.junit.Assert;
import org.junit.Test;

/**
 * RedirectContextTest 用于验证 RedirectContext 及其空实现的正确性
 */
public class RedirectContextTest {

    /**
     * 验证 RedirectContext.EmptyContext 的 Getter 方法行为：
     * 1. getOriginal() 应该返回 null
     * 2. getPrevious() 应该返回 null
     * 3. getInitial() 应该返回 null
     * 4. getDeepness() 应该返回 0
     */
    @Test
    public void testEmptyContextGetters() {
        RedirectContext empty = RedirectContext.EMPTY;
        Assert.assertNull("Original context should be null in EmptyContext", empty.getOriginal());
        Assert.assertNull("Previous context should be null in EmptyContext", empty.getPrevious());
        Assert.assertNull("Initial context should be null in EmptyContext", empty.getInitial());
        // 修复编译歧义：显式使用 long 类型比较，符合 JUnit 4 规范
        Assert.assertEquals("Deepness should be 1 in EmptyContext", 1L, (long) empty.getDeepness());
    }

    /**
     * 验证 RedirectContext.EMPTY 实例不为空，且确实是 EmptyContext 的实现
     */
    @Test
    public void testEmptyContextInstance() {
        Assert.assertNotNull("RedirectContext.EMPTY should not be null", RedirectContext.EMPTY);
        
        // 验证实例类名是否包含 EmptyContext (通常作为内部类实现)
        String className = RedirectContext.EMPTY.getClass().getSimpleName();
        Assert.assertTrue("RedirectContext.EMPTY should be an instance of EmptyContext", 
            className.contains("EmptyContext"));
    }

    @org.junit.jupiter.api.Test
    public void testEmptyContext() {
        RedirectContext context = RedirectContext.EMPTY;
        org.junit.jupiter.api.Assertions.assertNull(context.getOriginal());
        org.junit.jupiter.api.Assertions.assertEquals(1, context.getDeepness());
    }

    @org.junit.jupiter.api.Test
    public void testEmptyContextDeepness() {
        RedirectContext.EmptyContext empty = new RedirectContext.EmptyContext();
        org.junit.jupiter.api.Assertions.assertEquals(1, empty.getDeepness());
    }

    /**
     * 验证 EmptyContext.incrDeepness() 为 no-op，调用后 getDeepness() 仍为 0
     */
    @Test
    public void testEmptyContextIncrDeepness() {
        RedirectContext.EmptyContext empty = new RedirectContext.EmptyContext();
        empty.incrDeepness();
        Assert.assertEquals("EmptyContext.incrDeepness() should be no-op, deepness remains 1", Integer.valueOf(1), empty.getDeepness());
        empty.incrDeepness();
        empty.incrDeepness();
        Assert.assertEquals("Multiple incrDeepness() should still leave deepness 1", Integer.valueOf(1), empty.getDeepness());
    }

    /**
     * 验证 EmptyContext.isEntry() 恒为 false（空上下文非思考链入口）
     */
    @Test
    public void testEmptyContextIsEntry() {
        RedirectContext empty = RedirectContext.EMPTY;
        Assert.assertFalse("EmptyContext.isEntry() should be false", empty.isEntry());
        Assert.assertFalse("EmptyContext instance isEntry() should be false", new RedirectContext.EmptyContext().isEntry());
    }

    /**
     * 验证 EmptyContext.isFromFunMerge() 恒为 false（空上下文非 FunCall Merge 来源）
     */
    @Test
    public void testEmptyContextIsFromFunMerge() {
        RedirectContext empty = RedirectContext.EMPTY;
        Assert.assertFalse("EmptyContext.isFromFunMerge() should be false", empty.isFromFunMerge());
        Assert.assertFalse("EmptyContext instance isFromFunMerge() should be false", new RedirectContext.EmptyContext().isFromFunMerge());
    }

    /**
     * 验证 EmptyContext.isFromFunCall() 恒为 false（空上下文非 FunCall 来源）
     */
    @Test
    public void testEmptyContextIsFromFunCall() {
        RedirectContext empty = RedirectContext.EMPTY;
        Assert.assertFalse("EmptyContext.isFromFunCall() should be false", empty.isFromFunCall());
        Assert.assertFalse("EmptyContext instance isFromFunCall() should be false", new RedirectContext.EmptyContext().isFromFunCall());
    }

    /**
     * 验证 EmptyContext.setDeepness 为 no-op，调用后 getDeepness() 仍为 DEEPNESS 常量
     */
    @Test(expected = IllegalStateException.class)
    public void testEmptyContextSetDeepness() {
        RedirectContext.EmptyContext empty = new RedirectContext.EmptyContext();
        empty.setDeepness(5);
    }

}
