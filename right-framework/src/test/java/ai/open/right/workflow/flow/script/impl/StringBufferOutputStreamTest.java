package ai.open.right.workflow.flow.script.impl;

import org.junit.Assert;
import org.junit.Test;

import java.io.IOException;

public class StringBufferOutputStreamTest {

    @Test
    public void testWriter() throws IOException {
        PolyglotService.StringBufferOutputStream stringBufferOutputStream = new PolyglotService.StringBufferOutputStream();
        stringBufferOutputStream.write("A".getBytes());
        Assert.assertEquals("A", stringBufferOutputStream.getBuffer().toString());
        stringBufferOutputStream.write((byte) 97);
        Assert.assertEquals("Aa", stringBufferOutputStream.getBuffer().toString());
        stringBufferOutputStream.write("HELLO".getBytes(), 2, 3);
        Assert.assertEquals("AaLLO", stringBufferOutputStream.getBuffer().toString());
    }

    @Test
    public void testHashCode() throws Exception {
        Object object = PolyglotService.StringBufferOutputStream.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

}
