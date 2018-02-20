import React, {Component} from 'react';
import PropTypes from 'prop-types';
import Operational from './Operational.jsx'

class OperationalList extends Component{
    render() {
        return (
            <ul className="list-group">
                <li class="list-group-item list-group-item-success">Operational</li>{
                    this.props.operationals.map( up =>{
                    return <Operational
                        operational={up}
                        key={up.id}
                    />
                })
            }</ul>
        )
    }
}

OperationalList.propTypes = {
    operationals: PropTypes.array.isRequired
}

export default OperationalList
