import React, {Component} from 'react'
import PropTypes from 'prop-types';

class Outage extends Component {
    render() {
        const {outage} = this.props;
        return (
            <li className="list-group-item">
            <span className="badge"><span className="glyphicon glyphicon-remove" aria-hidden="true"></span>
            {outage.time} min</span>
                {outage.url}
            </li>
        )
    }
}

Outage.propTypes = {
    outage: PropTypes.object.isRequired
}

export default Outage
