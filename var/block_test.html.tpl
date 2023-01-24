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

    {% if not show_content2 %}
        not {{ content2 }}
    {% endif %}

    {% if show_content3 and show_content4 %}
        {{ content3 }} and {{ content4 }}
    {% endif %}

    {% if show_content3 and not show_content4 %}
        {{ content3 }} and not {{ content4 }}
    {% endif %}

    {% for k, v in list %}
        {{ k }}:{{ v }}
    {% endfor %}
{% include "./var/include_test.html.tpl" with PS(P("content4", content4)) only %}
{% endblock %}