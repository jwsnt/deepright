package ai.deepright.lang;


import java.util.*;

public class XmlResourceBundle extends ResourceBundle {

    protected final Properties properties;

    public XmlResourceBundle(Properties properties) {
        this.properties = properties;
    }

    @Override
    protected Object handleGetObject(String key) {
        return this.properties.getProperty(key);
    }

    @Override
    public Enumeration<String> getKeys() {
        return Collections.enumeration(this.properties.stringPropertyNames());
    }

    @Override
    public Set<String> handleKeySet() {
        return this.properties.stringPropertyNames();
    }
}
