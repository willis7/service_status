import React, {Component} from 'react';
import PropTypes from 'prop-types';
import Outage from './Outage.jsx'

class OutageList extends Component{
    render() {
        return (
            <ul className="list-group">
                <li className="list-group-item list-group-item-danger"><strong>Outage</strong></li>{
                    this.props.outages.map( out =>{
                    return <Outage
                        outage={out}
                        key={out.id}
                    />
                })
            }</ul>
        )
    }
}

OutageList.propTypes = {
    outages: PropTypes.array.isRequired
}

export default OutageList
