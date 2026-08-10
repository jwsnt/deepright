package ai.open.right.utils;

import com.fasterxml.jackson.dataformat.xml.XmlFactory;
import com.fasterxml.jackson.dataformat.xml.XmlMapper;
import com.fasterxml.jackson.dataformat.xml.ser.ToXmlGenerator;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;

import javax.xml.stream.XMLInputFactory;
import javax.xml.stream.XMLOutputFactory;
import java.io.InputStream;

@Slf4j
public class XmlUtils {

    public static final String PREFIX_XML = "```xml";

    public static final String SUFFIX = "```";

    public static XmlMapper MAPPER;

    static {
        // 禁用DTD和外部实体
        XMLInputFactory inputFactory = XMLInputFactory.newFactory();
        inputFactory.setProperty(XMLInputFactory.IS_SUPPORTING_EXTERNAL_ENTITIES, false);
        inputFactory.setProperty(XMLInputFactory.SUPPORT_DTD, false);
        XMLOutputFactory factory = XMLOutputFactory.newFactory();
        factory.setProperty(XMLOutputFactory.IS_REPAIRING_NAMESPACES, false);
        XmlUtils.MAPPER = new XmlMapper(new XmlFactory(inputFactory, factory));
        XmlUtils.MAPPER.configure(ToXmlGenerator.Feature.WRITE_XML_DECLARATION, true);
    }

    public static XmlMapper instance() {
        return XmlUtils.MAPPER;
    }

    public static <T> T read(InputStream t, Class<T> c) throws Exception {
        if (t != null) {
            try (InputStream input = t) {
                return XmlUtils.MAPPER.readValue(input, c);
            }
        } else {
            return null;
        }
    }

    public static <T> T read(String t, Class<T> c) throws Exception {
        if (t != null) {
            return XmlUtils.MAPPER.readValue(XmlUtils.clean(t), c);
        } else {
            return null;
        }
    }

    public static String write(Object t) throws Exception {
        return t == null ? null : (String.class.isAssignableFrom(t.getClass()) ? t.toString() : XmlUtils.MAPPER.writeValueAsString(t));
    }

    // 清理大模型产生的xml前后缀
    public static String clean(String t) throws Exception {
        String s = t.trim();
        if (StringUtils.startsWithIgnoreCase(s, XmlUtils.PREFIX_XML) && s.endsWith(XmlUtils.SUFFIX)) {
            s = new StringBuffer(s).delete(s.length() - XmlUtils.SUFFIX.length(), s.length()).delete(0, XmlUtils.PREFIX_XML.length()).toString();
        }
        s = s.trim();
        if (log.isDebugEnabled()) {
            log.debug("Clean xml={}", s);
        }
        return s;
    }
}
