{% extend "./var/base.html.tpl" %}
{% block content %}
    {{ __parent__ }}
    {% if show_content1 %}
        {{ content1 }}
    {% endif %}

    {% if show_content2 %}
        {{ content2 }}
    {% else %}
        not show content2
    {% endif %}

    {% if show_content3 %}
        {{ content3 }}
    {% elseif show_content4 %}
        {{ content4 }}
    {% endif %}

    {% for k, v in list %}
        {{ k }}:{{ v }}
    {% endfor %}
{% include "./var/include_test.html.tpl" with PS(P("content4", content4)) only %}
{% endblock %}